package svcoidc

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/middleware"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// ctxTenantHintKey 使用私有类型标记，避免与其它 context key 冲突。
type ctxKey string

// ctxTenantHintKey 承载 authorize 请求的租户跳转 hint（经 req.Context() 传递到 op.Storage）。
const ctxTenantHintKey ctxKey = "iam.tenantHint"

func RegisterProviderRoutes(routerGroup *gin.RouterGroup, provider *OIDCProvider, ssoSessionCookieName string) {
	var handler gin.HandlerFunc
	if provider != nil && provider.Provider != nil {
		handler = gin.WrapH(http.StripPrefix("/oidc", provider.Provider))
	} else {
		handler = func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		}
	}
	routerGroup.GET(oidc.DiscoveryEndpoint, handler)
	ssoStore := NewSSOSessionStore()
	silentAuth := middleware.SilentSSORequired("iam_sso_session", middleware.WithSessionValidator(func(ctx *gin.Context, sessionID string) error {
		_, err := ssoStore.ValidateSession(ctx.Request.Context(), sessionID)
		return err
	}))
	routerGroup.Any("/authorize", tenantHintMiddleware(), silentAuth, handler)
	routerGroup.Any("/authorize/callback", handler)
	routerGroup.Any("/oauth/token", handler)
	routerGroup.Any("/oauth/introspect", handler)
	routerGroup.Any("/userinfo", handler)
	routerGroup.Any("/revoke", handler)
	routerGroup.Any("/end_session", func(ctx *gin.Context) {
		ctx.SetCookie(ssoSessionCookieName, "", -1, "/", appconfig.Conf.OIDC.SSOCookieDomain(), false, true)
		handler(ctx)
	})
	routerGroup.Any("/keys", handler)
	routerGroup.Any("/healthz", handler)
	routerGroup.Any("/ready", handler)
}

// tenantHintMiddleware 读取 authorize 请求的 tenant query 参数并注入请求上下文，
// 供 op.Storage.CreateAuthRequest 经 req.Context() 读取；同时写入 gin 上下文供后续 SSO 使用。
func tenantHintMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if t := ctx.Query("tenant"); t != "" {
			if id, err := strconv.ParseUint(t, 10, 64); err == nil {
				withValue := context.WithValue(ctx.Request.Context(), ctxTenantHintKey, uint(id))
				ctx.Request = ctx.Request.WithContext(withValue)
				ctx.Set("tenantHint", uint(id))
			}
		}
		ctx.Next()
	}
}
