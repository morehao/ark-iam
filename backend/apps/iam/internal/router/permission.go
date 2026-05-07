package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrpermission"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func permissionRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrpermission.NewRoleCtr()
	menuCtr := ctrpermission.NewMenuCtr()
	scopeCtr := ctrpermission.NewScopeCtr()
	resourceCtr := ctrpermission.NewResourceCtr()
	roleMenuCtr := ctrpermission.NewRoleMenuCtr()
	roleScopeCtr := ctrpermission.NewRoleScopeCtr()
	userRoleCtr := ctrpermission.NewUserRoleCtr()
	appCtr := ctrpermission.NewApplicationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/permission/create", roleCtr.Create)
	v1RouterGroup.POST("/permission/delete", roleCtr.Delete)
	v1RouterGroup.POST("/permission/update", roleCtr.Update)
	v1RouterGroup.GET("/permission/detail", roleCtr.Detail)
	v1RouterGroup.POST("/permission/pageList", roleCtr.PageList)

	v1RouterGroup.POST("/permission/menu/create", menuCtr.Create)
	v1RouterGroup.POST("/permission/menu/delete", menuCtr.Delete)
	v1RouterGroup.POST("/permission/menu/update", menuCtr.Update)
	v1RouterGroup.GET("/permission/menu/detail", menuCtr.Detail)
	v1RouterGroup.POST("/permission/menu/pageList", menuCtr.PageList)
	v1RouterGroup.GET("/permission/menu/tree", menuCtr.Tree)

	v1RouterGroup.POST("/permission/scope/create", scopeCtr.Create)
	v1RouterGroup.POST("/permission/scope/delete", scopeCtr.Delete)
	v1RouterGroup.POST("/permission/scope/update", scopeCtr.Update)
	v1RouterGroup.GET("/permission/scope/detail", scopeCtr.Detail)
	v1RouterGroup.POST("/permission/scope/pageList", scopeCtr.PageList)

	v1RouterGroup.POST("/permission/resource/create", resourceCtr.Create)
	v1RouterGroup.POST("/permission/resource/delete", resourceCtr.Delete)
	v1RouterGroup.POST("/permission/resource/update", resourceCtr.Update)
	v1RouterGroup.GET("/permission/resource/detail", resourceCtr.Detail)
	v1RouterGroup.POST("/permission/resource/pageList", resourceCtr.PageList)

	v1RouterGroup.POST("/permission/role-menu/create", roleMenuCtr.Create)
	v1RouterGroup.POST("/permission/role-menu/delete", roleMenuCtr.Delete)
	v1RouterGroup.POST("/permission/role-menu/pageList", roleMenuCtr.PageList)

	v1RouterGroup.POST("/permission/role-scope/create", roleScopeCtr.Create)
	v1RouterGroup.POST("/permission/role-scope/delete", roleScopeCtr.Delete)
	v1RouterGroup.POST("/permission/role-scope/pageList", roleScopeCtr.PageList)

	v1RouterGroup.POST("/permission/user-role/create", userRoleCtr.Create)
	v1RouterGroup.POST("/permission/user-role/delete", userRoleCtr.Delete)
	v1RouterGroup.POST("/permission/user-role/pageList", userRoleCtr.PageList)

	v1RouterGroup.POST("/application/create", appCtr.Create)
	v1RouterGroup.POST("/application/delete", appCtr.Delete)
	v1RouterGroup.POST("/application/update", appCtr.Update)
	v1RouterGroup.GET("/application/detail", appCtr.Detail)
	v1RouterGroup.POST("/application/pageList", appCtr.PageList)
}