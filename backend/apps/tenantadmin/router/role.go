package router

import (
	"github.com/morehao/ark-iam/tenantadmin/controller/ctrtenant"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func roleRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	roleCtr := ctrtenant.NewRoleCtr(db)

	rg.POST("/roles", roleCtr.Create)
	rg.GET("/roles", roleCtr.PageList)
	rg.GET("/roles/:roleID", roleCtr.Detail)
	rg.PUT("/roles/:roleID", roleCtr.Update)
	rg.DELETE("/roles/:roleID", roleCtr.Delete)
	// 角色菜单授权（角色侧授权入口）
	rg.GET("/roles/:roleID/menus", roleCtr.GetMenus)
	rg.PUT("/roles/:roleID/menus", roleCtr.UpdateMenus)
}
