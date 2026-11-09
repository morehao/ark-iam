package rp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ClientCredentialsConfig 是 M2M（client_credentials）凭证配置。
//
// 用途：目录 API、introspect 等需要"应用自身身份"的调用。
// token_usage=machine 的令牌由 OP 签发，服务端据此识别机器主体。
type ClientCredentialsConfig struct {
	// TokenURL 是 OP 的 token 端点（如 http://iam:8081/oidc/oauth/token）。
	TokenURL string
	// ClientID / ClientSecret 是服务端客户端凭证（不要复用前端 PKCE 客户端）。
	ClientID     string
	ClientSecret string
	// Scopes 是请求的 scope（目录 API 需 directory.read）。
	Scopes []string
	// Resource 是 RFC 8707 resource 参数，决定令牌 aud；留空则 aud 为客户端自身标识。
	Resource string
}

// TokenClient 是 client_credentials 令牌的自动续期客户端。
//
// 续期策略与主流客户端一致：距 exp 不足 60s 即视为过期并换新，
// 避免边界时刻 401。
type TokenClient struct {
	cfg        ClientCredentialsConfig
	httpClient *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
	now         func() time.Time
}

// tokenExpiryLeeway 是 M2M 令牌的提前续期窗口。
const tokenExpiryLeeway = 60 * time.Second

// NewTokenClient 构造 M2M 令牌客户端。TokenURL/ClientID/ClientSecret 为空时报错。
func NewTokenClient(cfg ClientCredentialsConfig, hc *http.Client) (*TokenClient, error) {
	if cfg.TokenURL == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("rp: token client requires tokenURL, clientID and clientSecret")
	}
	if hc == nil {
		hc = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	return &TokenClient{cfg: cfg, httpClient: hc, now: time.Now}, nil
}

// Token 返回可用令牌，必要时自动续期。
func (c *TokenClient) Token(ctx context.Context) (string, error) {
	if c == nil {
		return "", errors.New("rp: nil token client")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessToken != "" && c.now().Add(tokenExpiryLeeway).Before(c.expiresAt) {
		return c.accessToken, nil
	}
	token, expiresIn, err := c.fetch(ctx)
	if err != nil {
		return "", err
	}
	c.accessToken = token
	c.expiresAt = c.now().Add(expiresIn)
	return token, nil
}

// Invalidate 丢弃缓存令牌（调用方收到 401 时可主动失效后重取）。
func (c *TokenClient) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.accessToken = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()
}

func (c *TokenClient) fetch(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	if len(c.cfg.Scopes) > 0 {
		form.Set("scope", strings.Join(c.cfg.Scopes, " "))
	}
	if c.cfg.Resource != "" {
		form.Set("resource", c.cfg.Resource)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(c.cfg.ClientID), url.QueryEscape(c.cfg.ClientSecret))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("rp: token endpoint returned %d: %s", resp.StatusCode, truncate(string(body), 256))
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", 0, fmt.Errorf("rp: decode token response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", 0, errors.New("rp: token response has no access_token")
	}
	ttl := time.Duration(payload.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return payload.AccessToken, ttl, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
