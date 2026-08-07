package platformadmin

import (
	"crypto/rsa"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/config"
	"github.com/morehao/ark-iam/platformadmin/internal/middleware/oidcauth"
	"github.com/morehao/ark-iam/platformadmin/internal/router"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

const AppName = "platformadmin"

func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: gconstant.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(config.Conf.JWT.SignKey, func() *rsa.PublicKey { return router.OIDCPublicKey }),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli, ginmiddleware.WithBlacklistKeyPrefix("platformadmin:token:blacklist:")),
		},
	})

	router.RegisterRouter(routerGroups, AppName)
}
