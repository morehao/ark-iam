package router

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/iam/internal/service/svcoidc"
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
		issuer = fmt.Sprintf("http://localhost:%s/v1/iam/oidc", port)
	}
	provider, err := svcoidc.SetupOIDCProvider(issuer)
	if err != nil {
		panic(err)
	}

	signingKey, err := provider.Storage.SigningKey(context.Background())
	if err != nil {
		panic(err)
	}
	OIDCPublicKey = &signingKey.Key().(*rsa.PrivateKey).PublicKey

	ctr := ctroidc.NewOIDCCtr(provider)
	ssoCookieDomain := config.Conf.OIDC.SSOCookieDomain()

	v1Group := groups.MustGetGroup(ginserver.ApiVersionV1)
	oidcGroup := v1Group.Group("/oidc")
	oidcGroup.Use(ginmiddleware.CORS())
	oidcGroup.POST("/login", ctr.Login)
	oidcGroup.GET("/sso-login", ctr.SSOLogin)
	oidcGroup.GET("/logged-out", func(ctx *gin.Context) {
		ctx.SetCookie("iam_sso_session", "", -1, "/", ssoCookieDomain, false, true)
		ctx.Redirect(302, config.Conf.OIDC.FrontendLoginURL)
	})
	svcoidc.RegisterProviderRoutes(oidcGroup, provider, "iam_sso_session")
}
