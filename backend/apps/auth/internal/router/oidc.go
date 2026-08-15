package router

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

var OIDCPublicKey *rsa.PublicKey

// InitOIDC 组装 OIDC provider 并注册全部 /oidc 协议路由。
// /oidc 为标准协议专用前缀（R3），不走业务路由规范，直接挂在 engine 上。
func InitOIDC(engine *gin.Engine, groups *ginserver.RouterGroups) {
	issuer := config.Conf.OIDC.Issuer
	if issuer == "" {
		port := config.Conf.Server.Port
		if port == "" {
			port = "8099"
		}
		issuer = fmt.Sprintf("http://localhost:%s/oidc", port)
	}
	provider, err := svcoidc.SetupOIDCProvider(issuer)
	if err != nil {
		panic(err)
	}

	signingKey, err := provider.Storage.SigningKey(context.Background())
	if err != nil {
		panic(err)
	}
	privKey := signingKey.Key().(*rsa.PrivateKey)
	OIDCPublicKey = &privKey.PublicKey

	// 启动 back-channel logout 发送器（异步消费登出队列）
	logoutWorker := svcoidc.NewLogoutWorker(privKey, signingKey.ID(), issuer)
	go logoutWorker.Run(context.Background())

	ctr := ctroidc.NewOIDCCtr(provider)
	// OIDC 协议端点（discovery/authorize/token/userinfo 等）为透传路由，统一复用该处理器
	oidcHandler := ctr.ProviderHandler()

	oidcGroup := engine.Group("/oidc")
	oidcGroup.Use(ginmiddleware.CORS())
	oidcGroup.POST("/login", ctr.Login)
	oidcGroup.POST("/login/selectTenant", ctr.SelectTenant)
	oidcGroup.GET("/sso-login", ctr.SSOLogin)
	oidcGroup.GET("/logged-out", ctr.LoggedOut)
	oidcGroup.GET(oidc.DiscoveryEndpoint, oidcHandler)
	// L1：prompt=none 静默登录失败跳回 redirect_uri 前，校验其确为该 client 注册的回调地址
	oidcGroup.Any("/authorize", middleware.TenantHint(), middleware.OIDCSilentAuth(provider, sso.SessionCookieName), oidcHandler)
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
