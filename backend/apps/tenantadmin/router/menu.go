package router

import (
	"github.com/morehao/ark-iam/backend/apps/tenantadmin/controller/ctrtenant"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// tenantMenuRouter 租户侧菜单路由。
// service 段 /v1/tenant 已与 /v1/platform/menus/tree 区分归属。
func tenantMenuRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	tenantMenuCtr := ctrtenant.NewTenantMenuCtr(db)

	rg.GET("/menus/tree", tenantMenuCtr.Tree)
}
