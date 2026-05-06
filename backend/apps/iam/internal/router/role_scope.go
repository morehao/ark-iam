package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrolescope"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleScopeRouter(groups *ginserver.RouterGroups) {
	roleScopeCtr := ctrrolescope.NewRoleScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/permission/create", roleScopeCtr.Create)
	v1RouterGroup.POST("/permission/delete", roleScopeCtr.Delete)
	v1RouterGroup.POST("/permission/pageList", roleScopeCtr.PageList)
}