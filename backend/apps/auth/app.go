package auth

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/middleware/oidcauth"
	"github.com/morehao/ark-iam/auth/internal/router"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

const AppName = "auth"

func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: gconstant.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(config.Conf.JWT.SignKey, func() *rsa.PublicKey { return router.OIDCPublicKey }, oidcauth.WithAuthSkipPaths(
				"/v1/auth/login",
				"/v1/auth/myTenants",
				"/v1/auth/selectTenant",
				"/v1/auth/register",
				"/v1/auth/refreshToken",
				"/v1/auth/connector/callback",
				"/v1/auth/oidc",
			)),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli, ginmiddleware.WithBlacklistKeyPrefix("auth:token:blacklist:")),
		},
	})

	router.RegisterRouter(routerGroups, AppName)
	router.InitOIDC(engine, routerGroups)
}
