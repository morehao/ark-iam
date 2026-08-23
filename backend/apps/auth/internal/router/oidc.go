package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// InitOIDC 注册全部 /oidc 协议路由。provider 的装配（签名密钥、logout worker 等）
// 由 ctroidc.NewOIDCCtr 自装配（router.RegisterRouter 内调用），这里只做路由注册。
// /oidc 为标准协议专用前缀（R3），不走业务路由规范，直接挂在 engine 上。
func InitOIDC(engine *gin.Engine, ctr *ctroidc.OIDCCtr) {
	// OIDC 协议端点（discovery/authorize/token/userinfo 等）为透传路由，统一复用该处理器
	oidcHandler := ctr.ProviderHandler()

	oidcGroup := engine.Group("/oidc")
	oidcGroup.Use(ginmiddleware.CORS())
	// 登录端点按 IP 频率限流（防暴力破解/CC，golib ratelimit；Redis 不可用时 fail-open）
	oidcGroup.POST("/login", middleware.LoginRateLimit(), ctr.Login)
	oidcGroup.POST("/login/selectTenant", ctr.SelectTenant)
	oidcGroup.POST("/registerPerson", ctr.RegisterPerson)
	oidcGroup.POST("/createTenant", ctr.CreateTenant)
	// login-config 为登录页前置策略查询（如是否允许自助注册/建租户），仅读协议态与应用策略
	oidcGroup.POST("/login-config", ctr.LoginConfig)
	oidcGroup.GET("/sso-login", ctr.SSOLogin)
	oidcGroup.GET("/logged-out", ctr.LoggedOut)
	oidcGroup.GET(oidc.DiscoveryEndpoint, oidcHandler)
	// L1：prompt=none 静默登录失败跳回 redirect_uri 前，校验其确为该 client 注册的回调地址
	// authorize 为 GET-only（OIDC Core §3.1.2.1）
	oidcGroup.GET("/authorize", middleware.TenantHint(), middleware.OIDCSilentAuth(ctr.Provider(), sso.SessionCookieName), oidcHandler)
	oidcGroup.Any("/authorize/callback", oidcHandler)
	// L3：token 端点读取 RFC 8707 resource 参数，供 storage 决定 access token 的 aud
	oidcGroup.Any("/oauth/token", middleware.ResourceHint(), oidcHandler)
	oidcGroup.Any("/oauth/introspect", oidcHandler)
	oidcGroup.Any("/userinfo", oidcHandler)
	oidcGroup.Any("/revoke", oidcHandler)
	oidcGroup.Any("/end_session", ctr.EndSession)
	oidcGroup.Any("/keys", oidcHandler)
	oidcGroup.Any("/healthz", oidcHandler)
	oidcGroup.Any("/ready", oidcHandler)
}
