package gateway

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth"
	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/platformadmin"
	"github.com/morehao/ark-iam/tenantadmin"
)

const AppName = "gateway"

func Init(engine *gin.Engine, Conf *config.Config) {
	auth.Init(engine, Conf)
	platformadmin.Init(engine, Conf)
	tenantadmin.Init(engine, Conf)
}
