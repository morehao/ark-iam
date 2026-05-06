package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrapplication"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func applicationRouter(groups *ginserver.RouterGroups) {
	appCtr := ctrapplication.NewApplicationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/application/create", appCtr.Create)
	v1RouterGroup.POST("/application/delete", appCtr.Delete)
	v1RouterGroup.POST("/application/update", appCtr.Update)
	v1RouterGroup.GET("/application/detail", appCtr.Detail)
	v1RouterGroup.POST("/application/pageList", appCtr.PageList)
}