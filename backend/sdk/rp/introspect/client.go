// Package introspect 是 RFC 7662（OAuth 2.0 Token Introspection）的最小客户端。
//
// 使用场景：需要"当前令牌是否已被撤销"的即时判定时（代价是多一次 IAM 调用，
// 属可选能力——默认鉴权路径应使用 sdk/rp 的本地验签）。
package introspect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config 是 introspection 客户端配置。
type Config struct {
	// Endpoint 是 OP 的 introspection 端点（如 http://iam:8081/oidc/oauth/introspect）。
	Endpoint string
	// ClientID / ClientSecret 是调用方（资源服务器）的 OIDC 客户端凭证。
	ClientID     string
	ClientSecret string
	// HTTPClient 可注入自定义客户端；默认 3s 超时。
	HTTPClient *http.Client
}

// Response 是 RFC 7662 的 introspection 响应（只解析常用字段）。
type Response struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope"`
	ClientID  string `json:"client_id"`
	TokenType string `json:"token_type"`
	Subject   string `json:"sub"`
	Username  string `json:"username"`
	// ExpiresAt / IssuedAt 是秒级时间戳（未提供时为 0）。
	ExpiresAt int64 `json:"exp"`
	IssuedAt  int64 `json:"iat"`
	// Claims 承载私有声明（tenant_id / user_id / token_usage / sid 等）。
	Claims map[string]any `json:"-"`
}

// Client 是 introspection 客户端。
type Client struct {
	cfg Config
	hc  *http.Client
}

// New 构造客户端。
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("introspect: endpoint, clientID and clientSecret are required")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{cfg: cfg, hc: hc}, nil
}

// Introspect 查询令牌状态。令牌无效/已撤销时 OP 返回 active=false（HTTP 200），不报错。
func (c *Client) Introspect(ctx context.Context, token string) (*Response, error) {
	form := url.Values{}
	form.Set("token", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspect: endpoint returned %d", resp.StatusCode)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("introspect: decode response: %w", err)
	}
	out := &Response{Claims: map[string]any{}}
	if v, ok := raw["active"].(bool); ok {
		out.Active = v
	}
	out.Scope, _ = raw["scope"].(string)
	out.ClientID, _ = raw["client_id"].(string)
	out.TokenType, _ = raw["token_type"].(string)
	out.Subject, _ = raw["sub"].(string)
	out.Username, _ = raw["username"].(string)
	if v, ok := raw["exp"].(float64); ok {
		out.ExpiresAt = int64(v)
	}
	if v, ok := raw["iat"].(float64); ok {
		out.IssuedAt = int64(v)
	}
	for _, key := range []string{"tenant_id", "user_id", "token_usage", "sid"} {
		if v, ok := raw[key]; ok {
			out.Claims[key] = v
		}
	}
	return out, nil
}
