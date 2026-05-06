package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrconnector"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrconnector.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/auth/create", connectorCtr.Create)
	v1RouterGroup.POST("/auth/delete", connectorCtr.Delete)
	v1RouterGroup.POST("/auth/update", connectorCtr.Update)
	v1RouterGroup.GET("/auth/detail", connectorCtr.Detail)
	v1RouterGroup.POST("/auth/pageList", connectorCtr.PageList)
}