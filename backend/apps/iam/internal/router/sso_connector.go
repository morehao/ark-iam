package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrsso_connector"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func ssoConnectorRouter(groups *ginserver.RouterGroups) {
	ssoConnectorCtr := ctrsso_connector.NewSsoConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/ssoConnector/create", ssoConnectorCtr.Create)
	v1RouterGroup.POST("/ssoConnector/delete", ssoConnectorCtr.Delete)
	v1RouterGroup.POST("/ssoConnector/update", ssoConnectorCtr.Update)
	v1RouterGroup.GET("/ssoConnector/detail", ssoConnectorCtr.Detail)
	v1RouterGroup.POST("/ssoConnector/pageList", ssoConnectorCtr.PageList)
}