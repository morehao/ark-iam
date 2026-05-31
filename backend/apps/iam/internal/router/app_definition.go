package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrappdefinition"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func appDefinitionRouter(groups *ginserver.RouterGroups) {
	ctr := ctrappdefinition.NewApplicationCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/app-definition/create", ctr.Create)
	v1RouterGroup.POST("/app-definition/delete", ctr.Delete)
	v1RouterGroup.POST("/app-definition/update", ctr.Update)
	v1RouterGroup.GET("/app-definition/detail", ctr.Detail)
	v1RouterGroup.GET("/app-definition/pageList", ctr.PageList)
}
