package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/router"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/golib/biz/gserver/gindocs"
)

const AppName = "auth"

func Init(engine *gin.Engine, Conf *pkgconfig.Config) {
	config.Conf = Conf

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	// 全部业务对象（OIDC provider、控制器、服务）由 router 层自装配，
	// app 层只做基础设施装配（配置、docs、引擎）。
	router.RegisterRouter(engine)
}
