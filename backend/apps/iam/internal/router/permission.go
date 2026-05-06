package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrpermission"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func permissionRouter(groups *ginserver.RouterGroups) {
	permissionCtr := ctrpermission.NewPermissionCtr()
	appCtr := ctrpermission.NewApplicationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/permission/create", permissionCtr.CreateRole)
	v1RouterGroup.POST("/permission/delete", permissionCtr.DeleteRole)
	v1RouterGroup.POST("/permission/update", permissionCtr.UpdateRole)
	v1RouterGroup.GET("/permission/detail", permissionCtr.DetailRole)
	v1RouterGroup.POST("/permission/pageList", permissionCtr.PageListRole)

	v1RouterGroup.POST("/permission/menu/create", permissionCtr.CreateMenu)
	v1RouterGroup.POST("/permission/menu/delete", permissionCtr.DeleteMenu)
	v1RouterGroup.POST("/permission/menu/update", permissionCtr.UpdateMenu)
	v1RouterGroup.GET("/permission/menu/detail", permissionCtr.DetailMenu)
	v1RouterGroup.POST("/permission/menu/pageList", permissionCtr.PageListMenu)
	v1RouterGroup.GET("/permission/menu/tree", permissionCtr.TreeMenu)

	v1RouterGroup.POST("/permission/scope/create", permissionCtr.CreateScope)
	v1RouterGroup.POST("/permission/scope/delete", permissionCtr.DeleteScope)
	v1RouterGroup.POST("/permission/scope/update", permissionCtr.UpdateScope)
	v1RouterGroup.GET("/permission/scope/detail", permissionCtr.DetailScope)
	v1RouterGroup.POST("/permission/scope/pageList", permissionCtr.PageListScope)

	v1RouterGroup.POST("/permission/resource/create", permissionCtr.CreateResource)
	v1RouterGroup.POST("/permission/resource/delete", permissionCtr.DeleteResource)
	v1RouterGroup.POST("/permission/resource/update", permissionCtr.UpdateResource)
	v1RouterGroup.GET("/permission/resource/detail", permissionCtr.DetailResource)
	v1RouterGroup.POST("/permission/resource/pageList", permissionCtr.PageListResource)

	v1RouterGroup.POST("/permission/role-menu/create", permissionCtr.CreateRoleMenu)
	v1RouterGroup.POST("/permission/role-menu/delete", permissionCtr.DeleteRoleMenu)
	v1RouterGroup.POST("/permission/role-menu/pageList", permissionCtr.PageListRoleMenu)

	v1RouterGroup.POST("/permission/role-scope/create", permissionCtr.CreateRoleScope)
	v1RouterGroup.POST("/permission/role-scope/delete", permissionCtr.DeleteRoleScope)
	v1RouterGroup.POST("/permission/role-scope/pageList", permissionCtr.PageListRoleScope)

	v1RouterGroup.POST("/permission/user-role/create", permissionCtr.CreateUserRole)
	v1RouterGroup.POST("/permission/user-role/delete", permissionCtr.DeleteUserRole)
	v1RouterGroup.POST("/permission/user-role/pageList", permissionCtr.PageListUserRole)

	v1RouterGroup.POST("/application/create", appCtr.Create)
	v1RouterGroup.POST("/application/delete", appCtr.Delete)
	v1RouterGroup.POST("/application/update", appCtr.Update)
	v1RouterGroup.GET("/application/detail", appCtr.Detail)
	v1RouterGroup.POST("/application/pageList", appCtr.PageList)
}
