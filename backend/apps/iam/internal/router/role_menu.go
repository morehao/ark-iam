package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrole_menu"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleMenuRouter(groups *ginserver.RouterGroups) {
	roleMenuCtr := ctrrole_menu.NewRoleMenuCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/roleMenu/create", roleMenuCtr.Create)
	v1RouterGroup.POST("/roleMenu/delete", roleMenuCtr.Delete)
	v1RouterGroup.POST("/roleMenu/pageList", roleMenuCtr.PageList)
}