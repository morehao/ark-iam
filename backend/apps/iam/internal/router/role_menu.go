package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrolemenu"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleMenuRouter(groups *ginserver.RouterGroups) {
	roleMenuCtr := ctrrolemenu.NewRoleMenuCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/rolemenu/create", roleMenuCtr.Create)
	v1RouterGroup.POST("/rolemenu/delete", roleMenuCtr.Delete)
	v1RouterGroup.POST("/rolemenu/pageList", roleMenuCtr.PageList)
}