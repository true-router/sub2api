// Package vmusers accesses vm2api's administrator-only subscription operations.
package vmusers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	Origin string
	token  string
	http   *http.Client
}

func Configured() (*Client, error) {
	origin := strings.TrimRight(os.Getenv("VM2API_INTEGRATION_URL"), "/")
	token := os.Getenv("VM2API_PANEL_TOKEN")
	u, err := url.Parse(origin)
	if err != nil || token == "" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, infraerrors.ServiceUnavailable("VM_NOT_CONFIGURED", "vm2api 集成地址或令牌未配置")
	}
	return &Client{Origin: origin, token: token, http: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

type Identity struct {
	Email    string `json:"email"`
	ID       string `json:"id"`
	Username string `json:"username"`
	Enabled  bool   `json:"enabled"`
	Role     string `json:"role"`
}
type Result struct {
	Item Identity `json:"item"`
	Key  string   `json:"default_key"`
}
type Slot struct {
	ID    string   `json:"slot_id"`
	Type  string   `json:"account_type"`
	Five  *float64 `json:"utilization_5h"`
	Seven *float64 `json:"utilization_7d"`
}
type Usage struct {
	Status string `json:"status"`
	Slots  []Slot `json:"slots"`
}

func (c *Client) request(ctx context.Context, method string, path string, body any, out any) error {
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Origin+"/api/panel/users"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return infraerrors.ServiceUnavailable("VM_DATA_UNAVAILABLE", "vm2api 连接失败")
	}
	defer res.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode/100 != 2 {
		if res.StatusCode == http.StatusConflict {
			return infraerrors.Conflict("VM_USER_CONFLICT", "vm2api 用户身份已变更或默认密钥不可用，请重新核对")
		}
		return infraerrors.New(http.StatusBadGateway, "VM_API_FAILED", "vm2api 管理接口不可用，请检查实例地址、版本和管理凭据")
	}
	if err := decoder.Decode(out); err != nil {
		return infraerrors.ServiceUnavailable("VM_DATA_UNAVAILABLE", "vm2api 响应格式无效")
	}
	return nil
}

func (c *Client) Lookup(ctx context.Context, id, name, email string) (*Identity, error) {
	query := url.Values{}
	if id != "" {
		query.Set("id", id)
	} else {
		query.Set("username", name)
		query.Set("email", email)
	}
	var out struct {
		Item *Identity `json:"item"`
	}
	if err := c.request(ctx, "GET", "/subscription-identity?"+query.Encode(), nil, &out); err != nil {
		return nil, err
	}
	if id != "" && (out.Item == nil || out.Item.ID != id) {
		return nil, infraerrors.ServiceUnavailable("VM_DATA_UNAVAILABLE", "已绑定的 vm2api 用户不存在")
	}
	if out.Item != nil && (out.Item.Role != "user" || (id == "" && !(strings.EqualFold(out.Item.Username, name) || (email != "" && strings.EqualFold(out.Item.Email, email))))) {
		return nil, infraerrors.BadRequest("VM_IDENTITY_INVALID", "只能关联身份匹配的普通用户")
	}
	return out.Item, nil
}

func (c *Client) Enable(ctx context.Context, user *Identity, name, email string) (Result, error) {
	var out Result
	var err error
	if user == nil {
		var secret [32]byte
		if _, err = rand.Read(secret[:]); err != nil {
			return out, err
		}
		var reply struct {
			Data Result `json:"data"`
		}
		err = c.request(ctx, "POST", "", map[string]string{"username": name, "email": email, "password": base64.RawURLEncoding.EncodeToString(secret[:]), "role": "user"}, &reply)
		out = reply.Data
	} else {
		err = c.request(ctx, "POST", "/"+url.PathEscape(user.ID)+"/activate-subscription", map[string]string{"username": user.Username, "email": email}, &out)
		if err == nil && out.Item.ID != user.ID {
			return Result{}, infraerrors.ServiceUnavailable("VM_DATA_UNAVAILABLE", "vm2api 用户身份已变更")
		}
	}
	if err == nil && (out.Item.ID == "" || (name != "" && !strings.EqualFold(out.Item.Username, name)) || (email != "" && !strings.EqualFold(out.Item.Email, email)) || !out.Item.Enabled || out.Item.Role != "user" || out.Key == "") {
		err = infraerrors.ServiceUnavailable("VM_DATA_UNAVAILABLE", "vm2api 用户身份或默认密钥无效")
	}
	return out, err
}

func (c *Client) Slots(ctx context.Context, id string) (*Usage, error) {
	var out struct {
		Count *int   `json:"slot_count"`
		Slots []Slot `json:"slots"`
	}
	if err := c.request(ctx, "GET", "/"+url.PathEscape(id)+"/slot-usage", nil, &out); err != nil {
		return nil, err
	}
	if out.Count == nil || out.Slots == nil || len(out.Slots) != *out.Count {
		return nil, infraerrors.ServiceUnavailable("VM_DATA_UNAVAILABLE", "vm2api 槽位列表无效")
	}
	return &Usage{Status: "available", Slots: out.Slots}, nil
}
