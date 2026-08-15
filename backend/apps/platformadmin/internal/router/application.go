package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrapplication"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func applicationRouter(groups *ginserver.RouterGroups) {
	ctr := ctrapplication.NewApplicationCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/applications", ctr.Create)
	v1RouterGroup.GET("/applications", ctr.PageList)
	v1RouterGroup.GET("/applications/:appID", ctr.Detail)
	v1RouterGroup.PUT("/applications/:appID", ctr.Update)
	v1RouterGroup.DELETE("/applications/:appID", ctr.Delete)
}
