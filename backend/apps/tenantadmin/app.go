package tenantadmin

import (
	"github.com/gin-gonic/gin"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware/oidcauth"
	"github.com/morehao/ark-iam/tenantadmin/config"
	"github.com/morehao/ark-iam/tenantadmin/internal/router"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

const AppName = "tenantadmin"

func Init(engine *gin.Engine, Conf *pkgconfig.Config) {
	config.Conf = Conf
	getOIDCPublicKey := router.InitOIDC()

	routerGroups := ginserver.NewRouterGroups(engine, "", ginserver.VersionGroup{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			oidcauth.OIDCCompatibleAuth(getOIDCPublicKey),
			ginmiddleware.TokenBlacklistCheck(dbclient.RedisCli,
				ginmiddleware.WithBlacklistKeyPrefix("tenantadmin:token:blacklist:")),
		},
	})

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	router.RegisterRouter(routerGroups, AppName)
}
