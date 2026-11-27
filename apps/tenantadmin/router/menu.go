package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/tenantadmin/controller/ctrtenant"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// tenantMenuRouter 租户侧菜单路由。
//   - /menus/tree：当前用户侧栏可见菜单树（只读）
//   - /applications/{appID}/menus...：归属当前租户的应用菜单维护（写 + 管理视图），
//     入口 requireSystemAdmin + 归属校验（见 svctenant.AppMenuSvc）
// service 段 /v1/tenant 已与 /v1/platform/menus/tree 区分归属。
func tenantMenuRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	tenantMenuCtr := ctrtenant.NewTenantMenuCtr(db)
	appMenuCtr := ctrtenant.NewAppMenuCtr(db)

	rg.GET("/menus/tree", tenantMenuCtr.Tree)

	rg.POST("/applications/:appID/menus", appMenuCtr.Create)
	rg.GET("/applications/:appID/menus/tree", appMenuCtr.Tree)
	rg.GET("/applications/:appID/menus/:menuID", appMenuCtr.Detail)
	rg.PUT("/applications/:appID/menus/:menuID", appMenuCtr.Update)
	rg.DELETE("/applications/:appID/menus/:menuID", appMenuCtr.Delete)
}
