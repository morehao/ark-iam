package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrsso"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func ssoConnectorRouter(groups *ginserver.RouterGroups) {
	ssoConnectorCtr := ctrsso.NewSsoConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/sso/create", ssoConnectorCtr.Create)
	v1RouterGroup.POST("/sso/delete", ssoConnectorCtr.Delete)
	v1RouterGroup.POST("/sso/update", ssoConnectorCtr.Update)
	v1RouterGroup.GET("/sso/detail", ssoConnectorCtr.Detail)
	v1RouterGroup.POST("/sso/pageList", ssoConnectorCtr.PageList)
}