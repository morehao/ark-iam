package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrconnector"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// connectorRouter 连接器管理面路由（平台级全局对象，/v1/platform）。
// 运行时授权/回调在 auth（/v1/auth/connectors/*），不在此注册。
func connectorRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	ctr := ctrconnector.NewConnectorCtr(db)
	rg.POST("/connectors", ctr.Create)
	rg.GET("/connectors", ctr.PageList)
	rg.GET("/connector-factories", ctr.GetFactoryList)
	rg.GET("/connectors/:connectorID", ctr.Detail)
	rg.PUT("/connectors/:connectorID", ctr.Update)
	rg.DELETE("/connectors/:connectorID", ctr.Delete)
}
