package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Request-local reuse prevents catalog rendering from repeating database reads
// for every model/protocol combination. No admission state crosses requests.
type unifiedLookupKey struct{}
type unifiedLookup struct {
	readOnly bool
	routes   map[int64][]CompositeModelRoute
	userID   int64
	loaded   bool
	groups   []Group
	subs     []UserSubscription
	accounts map[int64][]Account
	prepared map[int64]*UserSubscription
}

func withUnifiedLookup(ctx context.Context, userID int64) context.Context {
	if c, ok := ctx.Value(unifiedLookupKey{}).(*unifiedLookup); ok && c.userID == userID {
		return ctx
	}
	return context.WithValue(ctx, unifiedLookupKey{}, &unifiedLookup{userID: userID, routes: map[int64][]CompositeModelRoute{}, accounts: map[int64][]Account{}, prepared: map[int64]*UserSubscription{}})
}

const UnifiedAccessMode = "unified"

type unifiedKeyProvisioner interface {
	EnsureUnifiedAPIKey(context.Context, int64, string, bool, int) (int64, error)
}

type UnifiedWalletRoute struct {
	BillingModel string
	Endpoint     string
	Rate         float64
	Group        *Group
}

type UnifiedAPIModel struct {
	WalletRoutes          []UnifiedWalletRoute `json:"-"`
	ID                    string               `json:"id"`
	Platform              string               `json:"platform"`
	WalletGroupID         *int64               `json:"wallet_group_id,omitempty"`
	WalletRate            *float64             `json:"wallet_rate_multiplier,omitempty"`
	SubscriptionAvailable bool                 `json:"subscription_available"`
}

func (s *APIKeyService) SetUnifiedAccessDependencies(accounts AccountRepository, subscriptions *SubscriptionService, channels ChannelRepository, routes CompositeModelRouteRepository) {
	s.unifiedAccounts = accounts
	s.unifiedSubscriptions = subscriptions
	s.unifiedChannels = channels
	s.unifiedRoutes = routes
}

func (s *APIKeyService) EnsureUnifiedKey(ctx context.Context, userID int64, isDefault bool) (*APIKey, error) {
	if !isDefault {
		if err := s.checkAPIKeyCreateLimits(ctx, userID); err != nil {
			return nil, err
		}
	}
	repo, ok := s.apiKeyRepo.(unifiedKeyProvisioner)
	if !ok {
		return nil, fmt.Errorf("unified key persistence is unavailable")
	}
	credential, err := s.GenerateKey()
	if err != nil {
		return nil, err
	}
	limit := 0
	if s.cfg != nil {
		limit = s.cfg.APIKeyCreate.MaxActivePerUser
	}
	id, err := repo.EnsureUnifiedAPIKey(ctx, userID, credential, isDefault, limit)
	if err != nil {
		return nil, err
	}
	key, err := s.apiKeyRepo.GetByID(ctx, id)
	if errors.Is(err, ErrAPIKeyNotFound) {
		return nil, nil
	}
	return key, err
}

// resolveUnifiedGroup never changes the cached credential or its legacy peers.
// The returned request-owned key pins both routing and pricing through settlement.
func (s *APIKeyService) ResolveUnifiedRequest(ctx context.Context, key *APIKey, model, endpoint string) (*APIKey, *UserSubscription, error) {
	if key == nil || key.AccessMode != UnifiedAccessMode {
		return key, nil, nil
	}
	if strings.TrimSpace(model) == "" {
		return nil, nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required for unified API access")
	}
	ctx = withUnifiedLookup(ctx, key.UserID)
	groups, subs, err := s.unifiedGroups(ctx, key.UserID)
	if err != nil {
		return nil, nil, err
	}
	for i := range subs {
		sub := &subs[i]
		g := sub.Group
		if sub.DeletedAt != nil || g == nil || !g.IsActive() || time.Now().Before(sub.StartsAt) {
			continue
		}
		supports, e := s.unifiedGroupSupports(ctx, g, model, endpoint)
		if e != nil {
			return nil, nil, e
		}
		if !supports {
			continue
		}
		if s.unifiedSubscriptions == nil {
			return nil, nil, fmt.Errorf("subscription admission is unavailable")
		}
		// ListActiveUserSubscriptions reads persisted usage, avoiding stale L1 admission.
		cache, _ := ctx.Value(unifiedLookupKey{}).(*unifiedLookup)
		fresh := cache.prepared[sub.ID]
		if fresh == nil {
			if cache.readOnly {
				snapshot := *sub
				fresh = &snapshot
			} else {
				fresh, e = s.unifiedSubscriptions.EnsureWindowMaintenance(ctx, sub)
			}
			if e == nil {
				cache.prepared[sub.ID] = fresh
			}
		}
		if e != nil {
			return nil, nil, e
		}
		if _, e = s.unifiedSubscriptions.ValidateAndCheckLimits(fresh, g); e != nil {
			if errors.Is(e, ErrDailyLimitExceeded) || errors.Is(e, ErrWeeklyLimitExceeded) || errors.Is(e, ErrMonthlyLimitExceeded) || errors.Is(e, ErrSubscriptionExpired) {
				continue
			}
			return nil, nil, e
		}
		if (g.HasDailyLimit() && fresh.DailyUsageUSD >= *g.DailyLimitUSD) || (g.HasWeeklyLimit() && fresh.WeeklyUsageUSD >= *g.WeeklyLimitUSD) || (g.HasMonthlyLimit() && fresh.MonthlyUsageUSD >= *g.MonthlyLimitUSD) {
			continue
		}
		copy := *key
		copy.Group = g
		copy.GroupID = &g.ID
		return &copy, fresh, nil
	}
	for i := range groups {
		g := &groups[i]
		if g.IsSubscriptionType() || !key.User.CanBindGroup(g.ID, g.IsExclusive) {
			continue
		}
		supports, e := s.unifiedGroupSupports(ctx, g, model, endpoint)
		if e != nil {
			return nil, nil, e
		}
		if !supports {
			continue
		}
		copy := *key
		copy.Group = g
		copy.GroupID = &g.ID
		return &copy, nil, nil
	}
	return nil, nil, infraerrors.NotFound("MODEL_NOT_AVAILABLE", "No permitted route supports this model and API endpoint")
}

func (s *APIKeyService) unifiedGroups(ctx context.Context, userID int64) ([]Group, []UserSubscription, error) {
	cache, _ := ctx.Value(unifiedLookupKey{}).(*unifiedLookup)
	if cache != nil && cache.loaded {
		return cache.groups, cache.subs, nil
	}

	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, nil, err
	}
	subs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	// Stable wallet routing does not depend on current provider health or plan rate.
	rates, err := s.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(groups, func(i, j int) bool {
		_, left := rates[groups[i].ID]
		_, right := rates[groups[j].ID]
		if left != right {
			return left
		}
		return groups[i].ID < groups[j].ID
	})
	sort.Slice(subs, func(i, j int) bool {
		if subs[i].ExpiresAt.Equal(subs[j].ExpiresAt) {
			return subs[i].ID < subs[j].ID
		}
		return subs[i].ExpiresAt.Before(subs[j].ExpiresAt)
	})
	if cache != nil {
		cache.groups = groups
		cache.subs = subs
		cache.loaded = true
	}
	return groups, subs, nil
}

func (s *APIKeyService) unifiedGroupSupports(ctx context.Context, g *Group, model, endpoint string) (bool, error) {
	if !g.ModelAllowlist.Allows(model) {
		return false, nil
	}
	if endpoint == "messages" && g.Platform == PlatformOpenAI && !g.AllowMessagesDispatch {
		return false, nil
	}
	if s.unifiedAccounts == nil {
		return false, fmt.Errorf("model route discovery is unavailable")
	}
	accounts, err := s.unifiedGroupAccounts(ctx, g.ID)
	if err != nil {
		return false, err
	}
	lookupModel := model
	targetPlatform := ""
	if g.Platform == PlatformComposite && s.unifiedRoutes != nil {
		routes, e := s.unifiedGroupRoutes(ctx, g.ID)
		if e != nil {
			return false, e
		}
		if route, ok := matchCompositeRoute(routes, model, normalizeCompositeRouteEndpoint(endpoint)); ok {
			targetPlatform = route.TargetPlatform
			if route.UpstreamModel != "" {
				lookupModel = route.UpstreamModel
			}
		}
	}
	detected, known := DetectModelPlatform(lookupModel)
	for i := range accounts {
		a := &accounts[i]

		if g.Platform != PlatformComposite && a.Platform != g.Platform {
			continue
		}
		// A configured alias is authoritative; otherwise require a recognized family.
		if targetPlatform != "" && a.Platform != targetPlatform {
			continue
		}
		mapping := a.GetModelMapping()
		explicit := false
		for pattern := range mapping {
			if matchWildcard(pattern, lookupModel) {
				explicit = true
				break
			}
		}
		if !explicit && (!known || detected != a.Platform) {
			continue
		}
		if a.IsModelSupported(lookupModel) {
			return true, nil
		}
	}
	return false, nil
}

func (s *APIKeyService) unifiedGroupRoutes(ctx context.Context, groupID int64) ([]CompositeModelRoute, error) {
	if s.unifiedRoutes == nil {
		return nil, nil
	}
	cache, _ := ctx.Value(unifiedLookupKey{}).(*unifiedLookup)
	if cache != nil {
		if routes, ok := cache.routes[groupID]; ok {
			return routes, nil
		}
	}
	routes, err := s.unifiedRoutes.ListByGroup(ctx, groupID, false)
	if err == nil && cache != nil {
		cache.routes[groupID] = routes
	}
	return routes, err
}

func (s *APIKeyService) unifiedGroupAccounts(ctx context.Context, groupID int64) ([]Account, error) {
	cache, _ := ctx.Value(unifiedLookupKey{}).(*unifiedLookup)
	if cache != nil {
		if accounts, ok := cache.accounts[groupID]; ok {
			return accounts, nil
		}
	}
	accounts, err := s.unifiedAccounts.ListByGroup(ctx, groupID)
	if err == nil && cache != nil {
		cache.accounts[groupID] = accounts
	}
	return accounts, err
}

func (s *APIKeyService) UnifiedModels(ctx context.Context, userID int64) ([]UnifiedAPIModel, error) {
	ctx = withUnifiedLookup(ctx, userID)
	ctx.Value(unifiedLookupKey{}).(*unifiedLookup).readOnly = true
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups, subs, err := s.unifiedGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	allowedIDs := map[int64]bool{}
	for i := range groups {
		g := &groups[i]
		allowed := !g.IsSubscriptionType() && user.CanBindGroup(g.ID, g.IsExclusive)
		for _, sub := range subs {
			if sub.GroupID == g.ID {
				allowed = true
			}
		}
		if !allowed {
			continue
		}
		allowedIDs[g.ID] = true
		for _, p := range g.ModelPricing {
			for _, m := range p.Models {
				if !strings.Contains(m, "*") {
					platform := p.Platform
					if platform == "" {
						platform = g.Platform
					}
					names[m] = platform
				}
			}
		}
		routes, e := s.unifiedGroupRoutes(ctx, g.ID)
		if e != nil {
			return nil, e
		}
		for _, route := range routes {
			if route.Enabled && route.MatchType == CompositeRouteMatchExact {
				names[route.PublicModel] = route.TargetPlatform
			}
		}
		accounts, e := s.unifiedGroupAccounts(ctx, g.ID)
		if e != nil {
			return nil, e
		}
		for _, a := range accounts {

			for m := range a.GetModelMapping() {
				if !strings.Contains(m, "*") {
					names[m] = a.Platform
				}
			}
		}
	}
	if s.unifiedChannels != nil {
		channels, e := s.unifiedChannels.ListAll(ctx)
		if e != nil {
			return nil, e
		}
		for _, ch := range channels {
			visible := false
			for _, gid := range ch.GroupIDs {
				if allowedIDs[gid] {
					visible = true
					break
				}
			}
			if !visible {
				continue
			}
			if ch.Status != StatusActive {
				continue
			}
			for _, m := range ch.SupportedModels() {
				names[m.Name] = m.Platform
			}
		}
	}
	rates, err := s.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := []UnifiedAPIModel{}
	for name, platform := range names {
		row := UnifiedAPIModel{ID: name, Platform: platform}
		found := false
		for _, endpoint := range []string{"messages", "responses", "chat_completions"} {
			resolved, sub, e := s.ResolveUnifiedRequest(ctx, &APIKey{AccessMode: UnifiedAccessMode, UserID: userID, User: user}, name, endpoint)
			if e != nil {
				if infraerrors.IsNotFound(e) {
					continue
				}
				return nil, e
			}
			if sub != nil {
				row.SubscriptionAvailable = true
			}
			if resolved != nil {
				found = true
			}
			// Wallet price must be resolved independently of subscription priority.
			for i := range groups {
				g := &groups[i]
				if g.IsSubscriptionType() || !user.CanBindGroup(g.ID, g.IsExclusive) {
					continue
				}
				yes, e := s.unifiedGroupSupports(ctx, g, name, endpoint)
				if e != nil {
					return nil, e
				}
				if yes {
					rate := g.RateMultiplier
					if r, ok := rates[g.ID]; ok {
						rate = r
					}
					billingModel := name
					if g.Platform == PlatformComposite {
						routes, err := s.unifiedGroupRoutes(ctx, g.ID)
						if err != nil {
							return nil, err
						}
						if route, ok := matchCompositeRoute(routes, name, normalizeCompositeRouteEndpoint(endpoint)); ok && route.UpstreamModel != "" {
							billingModel = route.UpstreamModel
						}
					}
					row.WalletRoutes = append(row.WalletRoutes, UnifiedWalletRoute{BillingModel: billingModel, Endpoint: endpoint, Rate: rate * g.PeakMultiplierAt(time.Now()), Group: g})
					row.WalletGroupID = &g.ID
					row.WalletRate = &rate
					break
				}
			}
		}
		if found {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
