package router

import (
	"github.com/gin-gonic/gin"
	authconfig "github.com/morehao/ark-iam/apps/auth/config"
	"github.com/morehao/ark-iam/apps/auth/controller/ctroidc"
	"github.com/morehao/ark-iam/apps/auth/middleware"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// oidcCORS 返回 /oidc/* 协议端点的 CORS 中间件。
//
// /oidc/* 会读写 SSO 会话 cookie（iam_sso_session），跨站拓扑（cookieSameSite=none）
// 下浏览器必须跨源携带该 cookie。golib v1.36.2 起 CORS 默认不再下发
// Access-Control-Allow-Credentials（修掉了旧默认 Allow-Origin:* + Credentials:true 的
// 非法组合），因此这里显式开启凭证：配置了白名单则回显白名单内 Origin，否则动态回显
// 请求 Origin（allowCredentials 分支下不会再用通配 *）。
func oidcCORS() gin.HandlerFunc {
	opts := []ginmiddleware.CorsOption{ginmiddleware.WithAllowCredentials(true)}
	if authconfig.Conf != nil && len(authconfig.Conf.OIDC.CORSAllowOrigins) > 0 {
		opts = append(opts, ginmiddleware.WithAddAllowOrigins(authconfig.Conf.OIDC.CORSAllowOrigins...))
	}
	return ginmiddleware.CORS(opts...)
}

// InitOIDC 注册全部 /oidc 协议路由。provider 的装配（签名密钥、logout worker 等）
// 由 ctroidc.NewOIDCCtr 自装配（router.RegisterRouter 内调用），这里只做路由注册。
// /oidc 为标准协议专用前缀（R3），不走业务路由规范，直接挂在 engine 上。
func InitOIDC(engine *gin.RouterGroup, ctr *ctroidc.OIDCCtr) {
	// OIDC 协议端点（discovery/authorize/token/userinfo 等）为透传路由，统一复用该处理器
	oidcHandler := ctr.ProviderHandler()

	oidcGroup := engine.Group("/oidc")
	oidcGroup.Use(oidcCORS())
	// 登录端点按 IP 频率限流（防暴力破解/CC，golib ratelimit；Redis 不可用时 fail-open）
	oidcGroup.POST("/login", middleware.LoginRateLimit(), ctr.Login)
	oidcGroup.POST("/login/selectTenant", ctr.SelectTenant)
	// 首次登录强制改密（临时密码）：授权票据已绑定 subject，改密后需重新登录
	oidcGroup.POST("/login/changePassword", ctr.ChangePassword)
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
