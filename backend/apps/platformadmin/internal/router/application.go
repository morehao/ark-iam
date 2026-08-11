package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrapplication"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func applicationRouter(groups *ginserver.RouterGroups) {
	ctr := ctrapplication.NewApplicationCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/application/create", ctr.Create)
	v1RouterGroup.POST("/application/delete", ctr.Delete)
	v1RouterGroup.POST("/application/update", ctr.Update)
	v1RouterGroup.GET("/application/detail", ctr.Detail)
	v1RouterGroup.POST("/application/pageList", ctr.PageList)
}
