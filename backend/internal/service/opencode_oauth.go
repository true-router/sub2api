package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

const openCodeConsoleURL = "https://opencode.ai/console"
const openCodeOAuthClientID = "opencode-cli"

// OpenCodeOAuthService implements Console device authorization and token rotation.
// Endpoints are fixed: an imported credential cannot redirect credentials elsewhere.
type OpenCodeOAuthService struct {
	proxyRepo  ProxyRepository
	refreshAPI *OAuthRefreshAPI
}

func NewOpenCodeOAuthService(proxyRepo ProxyRepository, refreshAPI *OAuthRefreshAPI) *OpenCodeOAuthService {
	return &OpenCodeOAuthService{proxyRepo: proxyRepo, refreshAPI: refreshAPI}
}

type OpenCodeDeviceAuthorization struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri_complete"`
	ExpiresIn       int64  `json:"expires_in"`
	Interval        int64  `json:"interval"`
}

type openCodeOAuthToken struct {
	Access    string `json:"access_token"`
	Refresh   string `json:"refresh_token"`
	ExpiresIn int64  `json:"expires_in"`
	OrgID     string `json:"org_id"`
	Error     string `json:"error"`
}

func (s *OpenCodeOAuthService) request(ctx context.Context, proxyID *int64, method, path, token string, body, dest any) error {
	proxyURL := ""
	if proxyID != nil {
		if s.proxyRepo == nil {
			return fmt.Errorf("OpenCode OAuth proxy repository is unavailable")
		}
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err != nil {
			return fmt.Errorf("OpenCode OAuth proxy is unavailable")
		}
		proxyURL = proxy.URL()
	}
	client, err := httpclient.GetClient(httpclient.Options{ProxyURL: proxyURL, Timeout: 30 * time.Second})
	if err != nil {
		return fmt.Errorf("OpenCode OAuth transport is unavailable")
	}
	// Do not follow redirects with tokens or a refresh-token JSON body.
	bounded := *client
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, openCodeConsoleURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", openCodeUpstreamUserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := bounded.Do(req)
	if err != nil {
		return fmt.Errorf("OpenCode OAuth request failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("OpenCode OAuth response could not be read")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &failure)
		switch failure.Error {
		case "authorization_pending", "slow_down":
			if tokenResponse, ok := dest.(*openCodeOAuthToken); ok {
				tokenResponse.Error = failure.Error
				return nil
			}
		case "invalid_grant":
			return infraerrors.BadRequest("OPENCODE_REAUTH_REQUIRED", "OpenCode invalid_grant: authorize the account again")
		case "access_denied", "expired_token":
			return infraerrors.BadRequest("OPENCODE_AUTH_ENDED", "OpenCode device authorization was denied or expired")
		}
		return fmt.Errorf("OpenCode OAuth returned HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("OpenCode OAuth returned an invalid response")
	}
	return nil
}

func (s *OpenCodeOAuthService) Authorize(ctx context.Context, proxyID *int64) (*OpenCodeDeviceAuthorization, error) {
	var device OpenCodeDeviceAuthorization
	err := s.request(ctx, proxyID, http.MethodPost, "/auth/device/code", "", map[string]any{"client_id": openCodeOAuthClientID, "supports_org_scope": true}, &device)
	if err != nil {
		return nil, err
	}
	verification, err := url.Parse(device.VerificationURI)
	if err == nil {
		// Console returns relative URLs; resolve them as the official CLI does.
		base, _ := url.Parse(openCodeConsoleURL + "/")
		verification = base.ResolveReference(verification)
	}
	if err != nil || strings.TrimSpace(device.VerificationURI) == "" || verification.Scheme != "https" || verification.Host != "opencode.ai" || verification.User != nil || device.DeviceCode == "" || device.UserCode == "" || device.ExpiresIn <= 0 {
		return nil, fmt.Errorf("OpenCode returned invalid device authorization")
	}
	device.VerificationURI = verification.String()
	if device.Interval < 1 {
		device.Interval = 5
	}
	return &device, nil
}

// Poll performs one device-token exchange. The caller schedules polling using interval.
func (s *OpenCodeOAuthService) Poll(ctx context.Context, proxyID *int64, code string) (map[string]any, string, error) {
	var token openCodeOAuthToken
	err := s.request(ctx, proxyID, http.MethodPost, "/auth/device/token", "", map[string]any{"client_id": openCodeOAuthClientID, "grant_type": "urn:ietf:params:oauth:grant-type:device_code", "device_code": code}, &token)
	if err != nil {
		return nil, "", err
	}
	if token.Error == "authorization_pending" || token.Error == "slow_down" {
		return nil, token.Error, nil
	}
	credentials, err := openCodeTokenCredentials(token)
	if err != nil {
		return nil, "", err
	}
	var user struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	var orgs []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := s.request(ctx, proxyID, http.MethodGet, "/api/user", token.Access, nil, &user); err != nil {
		return nil, "", err
	}
	if err := s.request(ctx, proxyID, http.MethodGet, "/api/orgs", token.Access, nil, &orgs); err != nil {
		return nil, "", err
	}
	if token.OrgID == "" && len(orgs) == 1 {
		token.OrgID = orgs[0].ID
	}
	for _, org := range orgs {
		if org.ID == token.OrgID {
			credentials["org_id"] = org.ID
			credentials["org_name"] = org.Name
			credentials["account_id"] = user.ID
			credentials["email"] = user.Email
			return credentials, "authorized", nil
		}
	}
	return nil, "", infraerrors.BadRequest("OPENCODE_ORG_REQUIRED", "OpenCode authorization must select a valid workspace")
}

func openCodeTokenCredentials(token openCodeOAuthToken) (map[string]any, error) {
	if strings.TrimSpace(token.Access) == "" || strings.TrimSpace(token.Refresh) == "" || token.ExpiresIn <= 0 || token.ExpiresIn > 366*24*3600 {
		return nil, infraerrors.BadRequest("OPENCODE_TOKEN_INVALID", "OpenCode returned incomplete OAuth credentials")
	}
	return map[string]any{"access_token": token.Access, "refresh_token": token.Refresh, "expires_at": strconv.FormatInt(time.Now().Unix()+token.ExpiresIn, 10), "org_id": token.OrgID, "account_mode": AccountModeGo, "api_protocol": APIProtocolAdaptive}, nil
}

// Import accepts the native OpenCode credential JSON without reading arbitrary host files.
func (s *OpenCodeOAuthService) Import(raw json.RawMessage) (map[string]any, error) {
	var native struct {
		Type     string `json:"type"`
		Access   string `json:"access"`
		Refresh  string `json:"refresh"`
		Expires  int64  `json:"expires"`
		Metadata struct {
			Server    string `json:"server"`
			OrgID     string `json:"orgID"`
			OrgName   string `json:"orgName"`
			AccountID string `json:"accountID"`
			Email     string `json:"email"`
		} `json:"metadata"`
	}
	if json.Unmarshal(raw, &native) != nil || native.Type != "oauth" || (native.Metadata.Server != "" && native.Metadata.Server != openCodeConsoleURL && native.Metadata.Server != "https://console.opencode.ai") {
		return nil, infraerrors.BadRequest("OPENCODE_IMPORT_INVALID", "Paste an OpenCode Console OAuth credential object")
	}
	creds, err := openCodeTokenCredentials(openCodeOAuthToken{Access: native.Access, Refresh: native.Refresh, ExpiresIn: 3600})
	if err != nil {
		return nil, err
	}
	if native.Expires < 1_000_000_000_000 || native.Expires > 100_000_000_000_000 || native.Metadata.OrgID == "" {
		return nil, infraerrors.BadRequest("OPENCODE_IMPORT_INVALID", "OpenCode credential requires expires in milliseconds and metadata.orgID")
	}
	creds["expires_at"] = strconv.FormatInt(native.Expires/1000, 10)
	creds["org_id"] = native.Metadata.OrgID
	creds["org_name"] = native.Metadata.OrgName
	creds["account_id"] = native.Metadata.AccountID
	creds["email"] = native.Metadata.Email
	if err := ValidateOpenCodeOAuthCredentials(creds); err != nil {
		return nil, err
	}
	return creds, nil
}

func (s *OpenCodeOAuthService) CacheKey(account *Account) string {
	return fmt.Sprintf("opencode:oauth:%d", account.ID)
}
func (s *OpenCodeOAuthService) CanRefresh(account *Account) bool {
	return account.IsOpenCodeOAuth() && strings.TrimSpace(account.GetCredential("refresh_token")) != ""
}
func (s *OpenCodeOAuthService) NeedsRefresh(account *Account, window time.Duration) bool {
	if !s.CanRefresh(account) {
		return false
	}
	expiry := account.GetCredentialAsTime("expires_at")
	return account.GetCredential("access_token") == "" || expiry == nil || time.Until(*expiry) <= window
}
func (s *OpenCodeOAuthService) Refresh(ctx context.Context, account *Account) (map[string]any, error) {
	var token openCodeOAuthToken
	err := s.request(ctx, account.ProxyID, http.MethodPost, "/auth/device/token", "", map[string]any{"grant_type": "refresh_token", "refresh_token": account.GetCredential("refresh_token"), "client_id": openCodeOAuthClientID}, &token)
	if err != nil {
		return nil, err
	}
	creds, err := openCodeTokenCredentials(token)
	if err != nil {
		return nil, err
	}
	if token.OrgID == "" {
		delete(creds, "org_id")
	} else if token.OrgID != account.GetCredential("org_id") {
		creds["org_name"] = token.OrgID
	}
	delete(creds, "api_protocol")
	return MergeCredentials(account.Credentials, creds), nil
}

// Resolve serializes refresh with background/manual refresh and uses the durable result.
// Manual refresh uses the same shared lock as background and request refresh.
func (s *OpenCodeOAuthService) Resolve(ctx context.Context, account *Account, force bool) (*Account, error) {
	if s == nil || s.refreshAPI == nil {
		return nil, fmt.Errorf("OpenCode OAuth is unavailable")
	}
	window := 2 * time.Minute
	if force {
		window = 100 * 366 * 24 * time.Hour
	}
	for attempt := 0; attempt < 30; attempt++ {
		result, err := s.refreshAPI.RefreshIfNeeded(ctx, account, s, window)
		if err != nil {
			return nil, err
		}
		if !result.LockHeld {
			if result.Account == nil || !result.Account.IsOpenCodeOAuth() {
				return nil, fmt.Errorf("OpenCode OAuth account changed")
			}
			if expiry := result.Account.GetCredentialAsTime("expires_at"); expiry == nil || !expiry.After(time.Now()) {
				return nil, fmt.Errorf("OpenCode access token expired; authorize or refresh the account")
			}
			return result.Account, nil
		}
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("OpenCode OAuth refresh is busy; retry the request")
}

func (a *Account) IsOpenCodeOAuth() bool {
	return a != nil && a.Platform == PlatformOpenCodeGo && a.Type == AccountTypeOAuth
}

func ValidateOpenCodeOAuthCredentials(creds map[string]any) error {
	account := &Account{Platform: PlatformOpenCodeGo, Type: AccountTypeOAuth, Credentials: creds}
	if strings.TrimSpace(account.GetCredential("access_token")) == "" || strings.TrimSpace(account.GetCredential("refresh_token")) == "" || strings.TrimSpace(account.GetCredential("org_id")) == "" || account.GetCredentialAsTime("expires_at") == nil {
		return infraerrors.BadRequest("OPENCODE_CREDENTIALS_INVALID", "OpenCode OAuth requires access_token, refresh_token, expires_at and org_id")
	}
	if mode := account.GetCredential("account_mode"); mode != "" && mode != AccountModeGo {
		return infraerrors.BadRequest("OPENCODE_GO_ONLY", "OpenCode OAuth accounts use Go mode")
	}
	return nil
}

func (a *Account) openCodeOAuthBaseURL(protocol string) string {
	if !a.IsOpenCodeOAuth() {
		return ""
	}
	if protocol == APIProtocolAnthropic {
		return "https://opencode.ai/inference/go/anthropic"
	}
	return "https://opencode.ai/inference/go/openai/v1"
}

// applyOpenCodeOAuthHeaders runs after account overrides. Never trust inbound org/auth headers.
func applyOpenCodeOAuthHeaders(account *Account, headers http.Header) {
	if !account.IsOpenCodeOAuth() || headers == nil {
		return
	}
	for name := range headers {
		switch strings.ToLower(name) {
		case "authorization", "x-api-key", "x-goog-api-key", "x-org-id", "cookie":
			delete(headers, name)
		}
	}
	headers.Set("Authorization", "Bearer "+account.GetCredential("access_token"))
	headers.Set("X-Org-Id", account.GetCredential("org_id"))
}

// Console inference translates Gemini text models onto its standard API families.
func (a *Account) isUnsupportedOpenCodeModel(model string) bool {
	if !a.IsOpenCodeGo() {
		return false
	}
	if a.IsOpenCodeOAuth() && strings.HasPrefix(normalizeOpenCodeGoModelID(model), "gemini-") {
		return false
	}
	return IsOpenCodeUnsupportedModel(model)
}
