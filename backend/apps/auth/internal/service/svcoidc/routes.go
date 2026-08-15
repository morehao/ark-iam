package svcoidc

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// ctxKey 使用私有类型标记，避免与其它 context key 冲突。
type ctxKey string

// ctxTenantHintKey 承载 authorize 请求的租户跳转 hint（经 req.Context() 传递到 op.Storage）。
const ctxTenantHintKey ctxKey = "iam.tenantHint"

// ctxResourceKey 承载 token 端点的 RFC 8707 resource 参数（经 req.Context() 传递到 op.Storage）。
const ctxResourceKey ctxKey = "iam.resource"

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
	ssoStore := sso.NewSSOSessionStore()
	// L1：prompt=none 静默登录失败跳回 redirect_uri 前，校验其确为该 client 注册的回调地址，
	// 杜绝利用未注册 redirect_uri 构造开放重定向。
	silentAuth := middleware.SilentSSORequired(ssoSessionCookieName,
		middleware.WithSessionValidator(func(ctx *gin.Context, sessionID string) error {
			_, err := ssoStore.ValidateSession(ctx.Request.Context(), sessionID)
			return err
		}),
		middleware.WithRedirectURIVerifier(redirectURIVerifier(provider)),
	)
	routerGroup.Any("/authorize", tenantHintMiddleware(), silentAuth, handler)
	routerGroup.Any("/authorize/callback", handler)
	// L3：token 端点读取 RFC 8707 resource 参数，供 storage 决定 access token 的 aud。
	routerGroup.Any("/oauth/token", resourceHintMiddleware(), handler)
	routerGroup.Any("/oauth/introspect", handler)
	routerGroup.Any("/userinfo", handler)
	routerGroup.Any("/revoke", handler)
	routerGroup.Any("/end_session", func(ctx *gin.Context) {
		clearSSOCookie(ctx, ssoSessionCookieName)
		handler(ctx)
	})
	routerGroup.Any("/keys", handler)
	routerGroup.Any("/healthz", handler)
	routerGroup.Any("/ready", handler)
}

// redirectURIVerifier 返回基于 provider 客户端注册的回调地址校验器。
// provider 或其存储不可用（如单元测试中的空 provider）时返回 nil，跳过校验保持兼容。
func redirectURIVerifier(provider *OIDCProvider) func(ctx *gin.Context, clientID, redirectURI string) bool {
	if provider == nil || provider.Storage == nil || provider.Storage.persistentStore == nil {
		return nil
	}
	persistentStore := provider.Storage.persistentStore
	return func(ctx *gin.Context, clientID, redirectURI string) bool {
		if clientID == "" || redirectURI == "" {
			return false
		}
		client, err := persistentStore.GetClientByClientID(ctx.Request.Context(), clientID)
		if err != nil {
			return false
		}
		for _, u := range client.RedirectURIs() {
			if u == redirectURI {
				return true
			}
		}
		return false
	}
}

// tenantHintMiddleware 读取 authorize 请求的 tenant query 参数并注入请求上下文，
// 供 op.Storage.CreateAuthRequest 经 req.Context() 读取。
func tenantHintMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if t := ctx.Query("tenant"); t != "" {
			if id, err := strconv.ParseUint(t, 10, 64); err == nil {
				withValue := context.WithValue(ctx.Request.Context(), ctxTenantHintKey, uint(id))
				ctx.Request = ctx.Request.WithContext(withValue)
			}
		}
		ctx.Next()
	}
}

// resourceHintMiddleware 读取 token 端点的 RFC 8707 resource 参数并注入请求上下文，
// 供 op.Storage 决定 access token 的 aud。
func resourceHintMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if r := ctx.PostForm("resource"); r != "" {
			withValue := context.WithValue(ctx.Request.Context(), ctxResourceKey, r)
			ctx.Request = ctx.Request.WithContext(withValue)
		}
		ctx.Next()
	}
}

// ClearSSOCookie 依据配置清除 SSO 会话 cookie（Secure/SameSite 与写入时一致，L2）。
// 供 /oidc 路由组（end_session / logged-out）与其它登出路径复用。
func ClearSSOCookie(ctx *gin.Context, name string) {
	domain := ""
	secure := false
	sameSite := sso.DefaultSameSite
	if appconfig.Conf != nil {
		domain = appconfig.Conf.OIDC.SSOCookieDomain()
		secure = appconfig.Conf.OIDC.CookieSecure
		sameSite = appconfig.Conf.OIDC.CookieSameSiteMode()
	}
	sso.ClearSessionCookie(ctx, name, domain, secure, sameSite)
}

// clearSSOCookie 为包内调用提供简写。
func clearSSOCookie(ctx *gin.Context, name string) {
	ClearSSOCookie(ctx, name)
}
