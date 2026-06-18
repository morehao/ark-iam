package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrapplication"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func applicationRouter(groups *ginserver.RouterGroups) {
	ctr := ctrapplication.NewApplicationCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/application/create", ctr.Create)
	v1RouterGroup.POST("/platformadmin/application/delete", ctr.Delete)
	v1RouterGroup.POST("/platformadmin/application/update", ctr.Update)
	v1RouterGroup.GET("/platformadmin/application/detail", ctr.Detail)
	v1RouterGroup.POST("/platformadmin/application/pageList", ctr.PageList)
}
