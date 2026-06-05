package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/iam/internal/service/svcoidc"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

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

	ctr := ctroidc.NewOIDCCtr(provider)

	v1Group := groups.MustGetGroup(gconstant.ApiVersionV1)
	oidcGroup := v1Group.Group("/oidc")
	oidcGroup.Use(ginmiddleware.CORS())
	oidcGroup.POST("/login", ctr.Login)
	oidcGroup.GET("/sso-login", ctr.SSOLogin)
	oidcGroup.GET("/logged-out", func(ctx *gin.Context) {
		ctx.String(200, "You have been logged out.")
	})
	svcoidc.RegisterProviderRoutes(oidcGroup, provider)
}
