package iam

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	_ "github.com/morehao/ark-iam/iam/docs"
	"github.com/morehao/ark-iam/iam/internal/router"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

const AppName = "iam"

func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: gconstant.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			ginmiddleware.JWTAuth(config.Conf.JWT.SignKey, ginmiddleware.WithAuthSkipPaths(
				"/v1/iam/org/getConfigsByDomain",
				"/v1/iam/auth/register",
				"/v1/iam/auth/loginByPassword",
				"/v1/iam/auth/selectTenant",
				"/v1/iam/user/pageList",
			)),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli, ginmiddleware.WithBlacklistKeyPrefix("iam:token:blacklist:")),
		},
	})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	router.RegisterRouter(routerGroups, AppName)
}
