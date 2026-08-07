package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrauth"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/connector/create", connectorCtr.Create)
	v1RouterGroup.POST("/platformadmin/connector/delete", connectorCtr.Delete)
	v1RouterGroup.POST("/platformadmin/connector/update", connectorCtr.Update)
	v1RouterGroup.GET("/platformadmin/connector/detail", connectorCtr.Detail)
	v1RouterGroup.POST("/platformadmin/connector/pageList", connectorCtr.PageList)
	v1RouterGroup.POST("/platformadmin/connector/getFactoryList", connectorCtr.GetFactoryList)
	v1RouterGroup.POST("/platformadmin/connector/:connectorId/test", connectorCtr.TestConnector)
	v1RouterGroup.POST("/platformadmin/connector/:connectorId/authorize", connectorCtr.Authorize)
}
