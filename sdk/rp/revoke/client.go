// Package revoke 是 RFC 7009（OAuth 2.0 Token Revocation）的最小客户端。
//
// 用途：应用自身会话结束时撤销 refresh token / access token，不必只等 TTL。
// 对应 OP 侧已存在的端点：apps/auth 的 /oidc/revoke。
package revoke

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TokenTypeHint 是 RFC 7009 的 token_type_hint 取值。
type TokenTypeHint string

const (
	// TokenTypeRefreshToken 提示待撤销的是 refresh token。
	TokenTypeRefreshToken TokenTypeHint = "refresh_token"
	// TokenTypeAccessToken 提示待撤销的是 access token。
	TokenTypeAccessToken TokenTypeHint = "access_token"
)

// Config 是撤销客户端配置。
type Config struct {
	// Endpoint 是 OP 的撤销端点（如 http://iam:8081/oidc/revoke）。
	Endpoint string
	// ClientID / ClientSecret 是发起撤销的客户端凭证。
	ClientID     string
	ClientSecret string
	// HTTPClient 可注入自定义客户端；默认 3s 超时。
	HTTPClient *http.Client
}

// Client 是撤销客户端。
type Client struct {
	cfg Config
	hc  *http.Client
}

// New 构造客户端。
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("revoke: endpoint, clientID and clientSecret are required")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{cfg: cfg, hc: hc}, nil
}

// Revoke 撤销令牌。按 RFC 7009，未知或已失效的令牌也返回成功（幂等）。
func (c *Client) Revoke(ctx context.Context, token string, hint TokenTypeHint) error {
	if token == "" {
		return errors.New("revoke: empty token")
	}
	form := url.Values{}
	form.Set("token", token)
	if hint != "" {
		form.Set("token_type_hint", string(hint))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("revoke: endpoint returned %d", resp.StatusCode)
	}
	return nil
}
