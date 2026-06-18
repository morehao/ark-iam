package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctroauthclient"
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrpermission"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrpermission.NewRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/role/create", roleCtr.Create)
	v1RouterGroup.POST("/platformadmin/role/delete", roleCtr.Delete)
	v1RouterGroup.POST("/platformadmin/role/update", roleCtr.Update)
	v1RouterGroup.GET("/platformadmin/role/detail", roleCtr.Detail)
	v1RouterGroup.POST("/platformadmin/role/pageList", roleCtr.PageList)
	v1RouterGroup.GET("/platformadmin/role/users", roleCtr.ListUsers)
	v1RouterGroup.POST("/platformadmin/role/assignUsers", roleCtr.AssignUsers)
	v1RouterGroup.DELETE("/platformadmin/role/users/:roleId/:userId", roleCtr.RemoveUser)
}

func menuRouter(groups *ginserver.RouterGroups) {
	menuCtr := ctrpermission.NewMenuCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/menu/create", menuCtr.Create)
	v1RouterGroup.POST("/platformadmin/menu/delete", menuCtr.Delete)
	v1RouterGroup.POST("/platformadmin/menu/update", menuCtr.Update)
	v1RouterGroup.GET("/platformadmin/menu/detail", menuCtr.Detail)
	v1RouterGroup.POST("/platformadmin/menu/pageList", menuCtr.PageList)
	v1RouterGroup.GET("/platformadmin/menu/tree", menuCtr.Tree)
}

func scopeRouter(groups *ginserver.RouterGroups) {
	scopeCtr := ctrpermission.NewScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/scope/create", scopeCtr.Create)
	v1RouterGroup.POST("/platformadmin/scope/delete", scopeCtr.Delete)
	v1RouterGroup.POST("/platformadmin/scope/update", scopeCtr.Update)
	v1RouterGroup.GET("/platformadmin/scope/detail", scopeCtr.Detail)
	v1RouterGroup.POST("/platformadmin/scope/pageList", scopeCtr.PageList)
}

func resourceRouter(groups *ginserver.RouterGroups) {
	resourceCtr := ctrpermission.NewResourceCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/resource/create", resourceCtr.Create)
	v1RouterGroup.POST("/platformadmin/resource/delete", resourceCtr.Delete)
	v1RouterGroup.POST("/platformadmin/resource/update", resourceCtr.Update)
	v1RouterGroup.GET("/platformadmin/resource/detail", resourceCtr.Detail)
	v1RouterGroup.POST("/platformadmin/resource/pageList", resourceCtr.PageList)
}

func roleMenuRouter(groups *ginserver.RouterGroups) {
	roleMenuCtr := ctrpermission.NewRoleMenuCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/roleMenu/create", roleMenuCtr.Create)
	v1RouterGroup.POST("/platformadmin/roleMenu/delete", roleMenuCtr.Delete)
	v1RouterGroup.POST("/platformadmin/roleMenu/pageList", roleMenuCtr.PageList)
}

func roleScopeRouter(groups *ginserver.RouterGroups) {
	roleScopeCtr := ctrpermission.NewRoleScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/roleScope/create", roleScopeCtr.Create)
	v1RouterGroup.POST("/platformadmin/roleScope/delete", roleScopeCtr.Delete)
	v1RouterGroup.POST("/platformadmin/roleScope/pageList", roleScopeCtr.PageList)
}

func userRoleRouter(groups *ginserver.RouterGroups) {
	userRoleCtr := ctrpermission.NewUserRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/userRole/create", userRoleCtr.Create)
	v1RouterGroup.POST("/platformadmin/userRole/delete", userRoleCtr.Delete)
	v1RouterGroup.POST("/platformadmin/userRole/pageList", userRoleCtr.PageList)
}

func oauthClientRouter(groups *ginserver.RouterGroups) {
	appCtr := ctroauthclient.NewOAuthClientCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/oauthClient/create", appCtr.Create)
	v1RouterGroup.POST("/platformadmin/oauthClient/delete", appCtr.Delete)
	v1RouterGroup.POST("/platformadmin/oauthClient/update", appCtr.Update)
	v1RouterGroup.GET("/platformadmin/oauthClient/detail", appCtr.Detail)
	v1RouterGroup.POST("/platformadmin/oauthClient/pageList", appCtr.PageList)
	v1RouterGroup.GET("/platformadmin/oauthClient/secrets", appCtr.ListSecrets)
	v1RouterGroup.POST("/platformadmin/oauthClient/secrets", appCtr.CreateSecret)
	v1RouterGroup.DELETE("/platformadmin/oauthClient/secrets/:secretId", appCtr.DeleteSecret)
}
