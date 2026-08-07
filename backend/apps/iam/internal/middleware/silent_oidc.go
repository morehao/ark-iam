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

type silentSSOConfig struct {
	validate SSOValidator
}

// WithSessionValidator 注入 SSO 会话校验器。设置后，中间件会根据校验结果决定
// 是否放行，而不再仅凭 cookie 是否存在判定 SSO 会话有效。
func WithSessionValidator(v SSOValidator) func(*silentSSOConfig) {
	return func(c *silentSSOConfig) {
		c.validate = v
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
