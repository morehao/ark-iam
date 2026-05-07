package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrpermission"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrpermission.NewRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/role/create", roleCtr.Create)
	v1RouterGroup.POST("/role/delete", roleCtr.Delete)
	v1RouterGroup.POST("/role/update", roleCtr.Update)
	v1RouterGroup.GET("/role/detail", roleCtr.Detail)
	v1RouterGroup.POST("/role/pageList", roleCtr.PageList)
	v1RouterGroup.GET("/role/users", roleCtr.ListUsers)
	v1RouterGroup.POST("/role/assignUsers", roleCtr.AssignUsers)
	v1RouterGroup.DELETE("/role/users/:roleId/:userId", roleCtr.RemoveUser)
	v1RouterGroup.GET("/role/applications", roleCtr.ListApplications)
	v1RouterGroup.POST("/role/assignApplications", roleCtr.AssignApplications)
}

func menuRouter(groups *ginserver.RouterGroups) {
	menuCtr := ctrpermission.NewMenuCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/menu/create", menuCtr.Create)
	v1RouterGroup.POST("/menu/delete", menuCtr.Delete)
	v1RouterGroup.POST("/menu/update", menuCtr.Update)
	v1RouterGroup.GET("/menu/detail", menuCtr.Detail)
	v1RouterGroup.POST("/menu/pageList", menuCtr.PageList)
	v1RouterGroup.GET("/menu/tree", menuCtr.Tree)
}

func scopeRouter(groups *ginserver.RouterGroups) {
	scopeCtr := ctrpermission.NewScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/scope/create", scopeCtr.Create)
	v1RouterGroup.POST("/scope/delete", scopeCtr.Delete)
	v1RouterGroup.POST("/scope/update", scopeCtr.Update)
	v1RouterGroup.GET("/scope/detail", scopeCtr.Detail)
	v1RouterGroup.POST("/scope/pageList", scopeCtr.PageList)
}

func resourceRouter(groups *ginserver.RouterGroups) {
	resourceCtr := ctrpermission.NewResourceCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/resource/create", resourceCtr.Create)
	v1RouterGroup.POST("/resource/delete", resourceCtr.Delete)
	v1RouterGroup.POST("/resource/update", resourceCtr.Update)
	v1RouterGroup.GET("/resource/detail", resourceCtr.Detail)
	v1RouterGroup.POST("/resource/pageList", resourceCtr.PageList)
}

func roleMenuRouter(groups *ginserver.RouterGroups) {
	roleMenuCtr := ctrpermission.NewRoleMenuCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/roleMenu/create", roleMenuCtr.Create)
	v1RouterGroup.POST("/roleMenu/delete", roleMenuCtr.Delete)
	v1RouterGroup.POST("/roleMenu/pageList", roleMenuCtr.PageList)
}

func roleScopeRouter(groups *ginserver.RouterGroups) {
	roleScopeCtr := ctrpermission.NewRoleScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/roleScope/create", roleScopeCtr.Create)
	v1RouterGroup.POST("/roleScope/delete", roleScopeCtr.Delete)
	v1RouterGroup.POST("/roleScope/pageList", roleScopeCtr.PageList)
}

func userRoleRouter(groups *ginserver.RouterGroups) {
	userRoleCtr := ctrpermission.NewUserRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/userRole/create", userRoleCtr.Create)
	v1RouterGroup.POST("/userRole/delete", userRoleCtr.Delete)
	v1RouterGroup.POST("/userRole/pageList", userRoleCtr.PageList)
}

func applicationRouter(groups *ginserver.RouterGroups) {
	appCtr := ctrpermission.NewApplicationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/application/create", appCtr.Create)
	v1RouterGroup.POST("/application/delete", appCtr.Delete)
	v1RouterGroup.POST("/application/update", appCtr.Update)
	v1RouterGroup.GET("/application/detail", appCtr.Detail)
	v1RouterGroup.POST("/application/pageList", appCtr.PageList)
	v1RouterGroup.GET("/application/roles", appCtr.ListRoles)
	v1RouterGroup.POST("/application/assignRoles", appCtr.AssignRoles)
	v1RouterGroup.DELETE("/application/roles/:roleId", appCtr.RemoveRole)
}