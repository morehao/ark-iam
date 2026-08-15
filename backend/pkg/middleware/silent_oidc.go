package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	oidcPromptNone         = "none"
	oidcErrorLoginRequired = "login_required"
)

// SSOValidator 校验 SSO 会话标识是否仍然有效。命令者返回 nil 表示有效。
type SSOValidator func(ctx *gin.Context, sessionID string) error

// RedirectURIVerifier 校验 prompt=none 失败时要跳回的 redirect_uri 是否属于该 client。
// 返回 false 时中间件拒绝跳转（返回 400），防止开放重定向（L1）。
type RedirectURIVerifier func(ctx *gin.Context, clientID, redirectURI string) bool

type silentSSOConfig struct {
	validate          SSOValidator
	verifyRedirectURI RedirectURIVerifier
}

// WithSessionValidator 注入 SSO 会话校验器。设置后，中间件会根据校验结果决定
// 是否放行，而不再仅凭 cookie 是否存在判定 SSO 会话有效。
func WithSessionValidator(v SSOValidator) func(*silentSSOConfig) {
	return func(c *silentSSOConfig) {
		c.validate = v
	}
}

// WithRedirectURIVerifier 注入 redirect_uri 注册校验器。设置后，
// prompt=none 静默登录失败的错误跳转只允许落到该 client 注册的回调地址。
func WithRedirectURIVerifier(v RedirectURIVerifier) func(*silentSSOConfig) {
	return func(c *silentSSOConfig) {
		c.verifyRedirectURI = v
	}
}

func SilentSSORequired(ssoCookieName string, opts ...func(*silentSSOConfig)) gin.HandlerFunc {
	cfg := &silentSSOConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(ctx *gin.Context) {
		if ctx.Query("prompt") != oidcPromptNone {
			ctx.Next()
			return
		}
		if sessionID, err := ctx.Cookie(ssoCookieName); err == nil {
			if cfg.validate == nil || cfg.validate(ctx, sessionID) == nil {
				ctx.Next()
				return
			}
			// cookie 存在但对应 SSO 会话已失效/被撤销，按未登录处理，返回 login_required
		}

		redirectURI := ctx.Query("redirect_uri")
		if redirectURI == "" {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		u, err := url.Parse(redirectURI)
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		// L1：错误跳转只允许到该 client 注册的回调地址（防开放重定向）；
		// 未注入校验器（如仅测试场景）或校验失败时一律拒绝跳转。
		if cfg.verifyRedirectURI == nil || !cfg.verifyRedirectURI(ctx, ctx.Query("client_id"), redirectURI) {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		state := ctx.Query("state")
		q := u.Query()
		q.Set("error", oidcErrorLoginRequired)
		if state != "" {
			q.Set("state", state)
		}
		u.RawQuery = q.Encode()
		ctx.Redirect(http.StatusFound, u.String())
		ctx.Abort()
	}
}
