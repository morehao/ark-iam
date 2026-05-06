package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrresource"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func resourceRouter(groups *ginserver.RouterGroups) {
	resourceCtr := ctrresource.NewResourceCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/resource/create", resourceCtr.Create)
	v1RouterGroup.POST("/resource/delete", resourceCtr.Delete)
	v1RouterGroup.POST("/resource/update", resourceCtr.Update)
	v1RouterGroup.GET("/resource/detail", resourceCtr.Detail)
	v1RouterGroup.POST("/resource/pageList", resourceCtr.PageList)
}