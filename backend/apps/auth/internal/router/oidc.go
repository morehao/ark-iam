package router

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

var OIDCPublicKey *rsa.PublicKey

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

	oidcGroup := engine.Group("/oidc")
	oidcGroup.Use(ginmiddleware.CORS())
	oidcGroup.POST("/login", ctr.Login)
	oidcGroup.POST("/login/selectTenant", ctr.SelectTenant)
	oidcGroup.GET("/sso-login", ctr.SSOLogin)
	oidcGroup.GET("/logged-out", func(ctx *gin.Context) {
		svcoidc.ClearSSOCookie(ctx, sso.SessionCookieName)
		ctx.Redirect(302, config.Conf.OIDC.FrontendLoginURL)
	})
	svcoidc.RegisterProviderRoutes(oidcGroup, provider, sso.SessionCookieName)
}
