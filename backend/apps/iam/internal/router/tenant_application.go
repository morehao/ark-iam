package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrtenantapplication"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantApplicationRouter(groups *ginserver.RouterGroups) {
	ctr := ctrtenantapplication.NewTenantApplicationCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenantApplication/create", ctr.Create)
	v1RouterGroup.POST("/tenantApplication/delete", ctr.Delete)
	v1RouterGroup.POST("/tenantApplication/update", ctr.Update)
	v1RouterGroup.GET("/tenantApplication/detail", ctr.Detail)
	v1RouterGroup.POST("/tenantApplication/pageList", ctr.PageList)
}
