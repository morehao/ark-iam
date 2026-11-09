// Package userinfo 是 OIDC UserInfo 端点（OIDC Core §5.3）的最小客户端。
//
// 用途：取当前登录人的资料（姓名/头像/邮箱/电话）与角色编码（groups）。
// 按设计文档纪律：**userinfo 不得出现在鉴权或审计写入的关键路径上**——
// 审计姓名用写入时快照，取不到就只记 id。
package userinfo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config 是 userinfo 客户端配置。
type Config struct {
	// Endpoint 是 OP 的 userinfo 端点（如 http://iam:8081/oidc/userinfo）。
	Endpoint string
	// HTTPClient 可注入自定义客户端；默认 3s 超时。
	HTTPClient *http.Client
}

// Info 是 userinfo 响应（标准声明 + 本仓的 groups 扩展）。
type Info struct {
	Subject           string   `json:"sub"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	PhoneNumber       string   `json:"phone_number"`
	Picture           string   `json:"picture"`
	Groups            []string `json:"groups"`
	// Extra 承载未在结构体里显式声明的声明（前向兼容：忽略未知字段不报错）。
	Extra map[string]any `json:"-"`
}

// Client 是 userinfo 客户端。
type Client struct {
	cfg Config
	hc  *http.Client
}

// New 构造客户端。
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("userinfo: endpoint is required")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{cfg: cfg, hc: hc}, nil
}

// Get 用 access token 取 userinfo。
func (c *Client) Get(ctx context.Context, accessToken string) (*Info, error) {
	if accessToken == "" {
		return nil, errors.New("userinfo: empty access token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
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
		return nil, fmt.Errorf("userinfo: endpoint returned %d", resp.StatusCode)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("userinfo: decode response: %w", err)
	}
	out := &Info{Extra: map[string]any{}}
	readString := func(key string) string {
		v, _ := raw[key].(string)
		return v
	}
	out.Subject = readString("sub")
	out.Name = readString("name")
	out.PreferredUsername = readString("preferred_username")
	out.Email = readString("email")
	out.PhoneNumber = readString("phone_number")
	out.Picture = readString("picture")
	if v, ok := raw["email_verified"].(bool); ok {
		out.EmailVerified = v
	}
	if list, ok := raw["groups"].([]any); ok {
		for _, item := range list {
			if s, ok := item.(string); ok && s != "" {
				out.Groups = append(out.Groups, s)
			}
		}
	}
	for k, v := range raw {
		switch k {
		case "sub", "name", "preferred_username", "email", "email_verified", "phone_number", "picture", "groups":
			continue
		}
		out.Extra[k] = v
	}
	return out, nil
}

// DisplayName 返回可用于审计快照的展示名（优先 name，回退 preferred_username）。
// 取不到时返回空串——调用方必须只记 id，绝不因缺姓名丢审计行。
func (i *Info) DisplayName() string {
	if i == nil {
		return ""
	}
	if strings.TrimSpace(i.Name) != "" {
		return i.Name
	}
	return i.PreferredUsername
}
