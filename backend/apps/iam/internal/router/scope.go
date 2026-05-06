package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrscope"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func scopeRouter(groups *ginserver.RouterGroups) {
	scopeCtr := ctrscope.NewScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/resource/create", scopeCtr.Create)
	v1RouterGroup.POST("/resource/delete", scopeCtr.Delete)
	v1RouterGroup.POST("/resource/update", scopeCtr.Update)
	v1RouterGroup.GET("/resource/detail", scopeCtr.Detail)
	v1RouterGroup.POST("/resource/pageList", scopeCtr.PageList)
}