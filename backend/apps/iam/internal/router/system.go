package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrsystem"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func systemRouter(groups *ginserver.RouterGroups) {
	systemCtr := ctrsystem.NewSystemCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/system/create", systemCtr.Create)
	v1RouterGroup.POST("/system/delete", systemCtr.Delete)
	v1RouterGroup.POST("/system/update", systemCtr.Update)
	v1RouterGroup.GET("/system/detail", systemCtr.Detail)
	v1RouterGroup.POST("/system/pageList", systemCtr.PageList)
}