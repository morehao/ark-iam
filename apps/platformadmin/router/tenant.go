package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrtenant"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func tenantRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	tenantCtr := ctrtenant.NewTenantCtr(db)
	rg.POST("/tenants", tenantCtr.Create)
	rg.GET("/tenants", tenantCtr.PageList)
	rg.GET("/tenants/:tenantID", tenantCtr.Detail)
	rg.PUT("/tenants/:tenantID", tenantCtr.Update)
	rg.DELETE("/tenants/:tenantID", tenantCtr.Delete)
	// offboard：P0 D 唯一停用入口（tenant suspended + primary disabled + 清 issuer + 撤 session）
	rg.POST("/tenants/:tenantID/offboard", tenantCtr.Offboard)
	// console-binding 管理（Host→租户/issuer/client；与租户同域挂载）
	consoleBindingRouter(rg, db)
	// 重置内置管理员密码（R2 动作子路径；builtin-admin 为租户下的单体子资源）
	rg.POST("/tenants/:tenantID/builtin-admin/reset-password", tenantCtr.ResetAdminPassword)
}
