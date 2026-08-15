package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrtenant.NewRoleCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/roles", roleCtr.Create)
	v1RouterGroup.GET("/roles", roleCtr.PageList)
	v1RouterGroup.GET("/roles/:roleID", roleCtr.Detail)
	v1RouterGroup.PUT("/roles/:roleID", roleCtr.Update)
	v1RouterGroup.DELETE("/roles/:roleID", roleCtr.Delete)
	// 角色菜单授权（角色侧授权入口）
	v1RouterGroup.GET("/roles/:roleID/menus", roleCtr.GetMenus)
	v1RouterGroup.PUT("/roles/:roleID/menus", roleCtr.UpdateMenus)
}
