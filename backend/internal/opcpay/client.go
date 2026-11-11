package opcpay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

const BaseURL = "https://ai6666.com/opcpay/api/pay/v1"

// Client implements the same server-only contract as opcpay 0.3.4.
type Client struct {
	HTTP *http.Client
	Base string
	key  string
}

func NewClient(apiKey string) *Client {
	return &Client{HTTP: &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Base: BaseURL, key: apiKey}
}
func (c *Client) Configured() bool { return c.key != "" }
func (c *Client) Call(ctx context.Context, path string, body any, result any) error {
	if !c.Configured() {
		return errors.New("一人付尚未配置")
	}
	method := http.MethodGet
	var data []byte
	var err error
	if body != nil {
		method = http.MethodPost
		data, err = json.Marshal(body)
		if err != nil {
			return errors.New("一人付请求无效")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("一人付请求无效")
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New("一人付请求未确认，请恢复原操作")
	}
	defer resp.Body.Close()
	// Never include upstream bodies or transport errors in logs or responses.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("一人付请求未确认，请恢复原操作")
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result) != nil {
		return errors.New("一人付响应无效，请恢复原操作")
	}
	return nil
}

type Capabilities struct {
	ProviderConfigured bool `json:"provider_configured"`
	Flags              struct {
		Purchase bool `json:"purchase_entry"`
		Access   bool `json:"entitlement_query"`
		Consume  bool `json:"consumption"`
	} `json:"capabilities"`
}
type Option struct {
	SKU      string  `json:"sku_id"`
	Name     string  `json:"display_name"`
	Mode     string  `json:"billing_mode"`
	Price    string  `json:"price"`
	Currency string  `json:"currency"`
	Unit     string  `json:"quota_unit,omitempty"`
	Quantity float64 `json:"quota_amount,omitempty"`
}
type Product struct {
	Status  string   `json:"capability_status"`
	Name    string   `json:"product_name"`
	Mode    string   `json:"billing_mode"`
	Unit    string   `json:"quota_unit,omitempty"`
	Options []Option `json:"charging_options"`
}
type Access struct {
	Granted             bool    `json:"access_granted"`
	Decision            string  `json:"decision"`
	RequiresConsumption bool    `json:"requires_consumption"`
	Remaining           float64 `json:"remaining"`
	Unit                string  `json:"quota_unit,omitempty"`
	ValidUntil          string  `json:"valid_until,omitempty"`
}
type Result struct {
	Status    string  `json:"status"`
	Confirmed bool    `json:"confirmed"`
	Key       string  `json:"idempotency_key"`
	URL       string  `json:"purchase_url,omitempty"`
	QR        string  `json:"purchase_qr_code,omitempty"`
	Consumed  float64 `json:"consumed"`
	Unit      string  `json:"quota_unit,omitempty"`
}
