package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrpermission"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrpermission.NewRoleCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/roles", roleCtr.Create)
	v1RouterGroup.GET("/roles", roleCtr.PageList)
	v1RouterGroup.GET("/roles/:roleID", roleCtr.Detail)
	v1RouterGroup.PUT("/roles/:roleID", roleCtr.Update)
	v1RouterGroup.DELETE("/roles/:roleID", roleCtr.Delete)
	// 角色-用户关联
	v1RouterGroup.GET("/roles/:roleID/users", roleCtr.ListUsers)
	v1RouterGroup.PUT("/roles/:roleID/users", roleCtr.AssignUsers)
	v1RouterGroup.DELETE("/roles/:roleID/users/:userID", roleCtr.RemoveUser)
}

func menuRouter(groups *ginserver.RouterGroups) {
	menuCtr := ctrpermission.NewMenuCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/menus", menuCtr.Create)
	v1RouterGroup.GET("/menus", menuCtr.PageList)
	v1RouterGroup.GET("/menus/tree", menuCtr.Tree)
	v1RouterGroup.GET("/menus/:menuID", menuCtr.Detail)
	v1RouterGroup.PUT("/menus/:menuID", menuCtr.Update)
	v1RouterGroup.DELETE("/menus/:menuID", menuCtr.Delete)
}

func scopeRouter(groups *ginserver.RouterGroups) {
	scopeCtr := ctrpermission.NewScopeCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/scopes", scopeCtr.Create)
	v1RouterGroup.GET("/scopes", scopeCtr.PageList)
	v1RouterGroup.GET("/scopes/:scopeID", scopeCtr.Detail)
	v1RouterGroup.PUT("/scopes/:scopeID", scopeCtr.Update)
	v1RouterGroup.DELETE("/scopes/:scopeID", scopeCtr.Delete)
}

func resourceRouter(groups *ginserver.RouterGroups) {
	resourceCtr := ctrpermission.NewResourceCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/resources", resourceCtr.Create)
	v1RouterGroup.GET("/resources", resourceCtr.PageList)
	v1RouterGroup.GET("/resources/:resourceID", resourceCtr.Detail)
	v1RouterGroup.PUT("/resources/:resourceID", resourceCtr.Update)
	v1RouterGroup.DELETE("/resources/:resourceID", resourceCtr.Delete)
}

func roleMenuRouter(groups *ginserver.RouterGroups) {
	roleMenuCtr := ctrpermission.NewRoleMenuCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/roles/:roleID/menus", roleMenuCtr.PageList)
	v1RouterGroup.POST("/roles/:roleID/menus", roleMenuCtr.Create)
	v1RouterGroup.DELETE("/roles/:roleID/menus/:menuID", roleMenuCtr.Delete)
}

func roleScopeRouter(groups *ginserver.RouterGroups) {
	roleScopeCtr := ctrpermission.NewRoleScopeCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/roles/:roleID/scopes", roleScopeCtr.PageList)
	v1RouterGroup.POST("/roles/:roleID/scopes", roleScopeCtr.Create)
	v1RouterGroup.DELETE("/roles/:roleID/scopes/:scopeID", roleScopeCtr.Delete)
}

func userRoleRouter(groups *ginserver.RouterGroups) {
	userRoleCtr := ctrpermission.NewUserRoleCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/users/:userID/roles", userRoleCtr.PageList)
	v1RouterGroup.POST("/users/:userID/roles", userRoleCtr.Create)
	v1RouterGroup.DELETE("/users/:userID/roles/:roleID", userRoleCtr.Delete)
}

func applicationClientRouter(groups *ginserver.RouterGroups) {
	appCtr := ctrapplicationclient.NewApplicationClientCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/application-clients", appCtr.Create)
	v1RouterGroup.GET("/application-clients", appCtr.PageList)
	v1RouterGroup.GET("/application-clients/:applicationClientID", appCtr.Detail)
	v1RouterGroup.PUT("/application-clients/:applicationClientID", appCtr.Update)
	v1RouterGroup.DELETE("/application-clients/:applicationClientID", appCtr.Delete)
	// 客户端密钥（子资源）
	v1RouterGroup.GET("/application-clients/:applicationClientID/secrets", appCtr.ListSecrets)
	v1RouterGroup.POST("/application-clients/:applicationClientID/secrets", appCtr.CreateSecret)
	v1RouterGroup.DELETE("/application-clients/:applicationClientID/secrets/:secretID", appCtr.DeleteSecret)
}
