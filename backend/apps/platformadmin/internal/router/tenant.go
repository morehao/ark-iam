package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/tenants", tenantCtr.Create)
	v1RouterGroup.GET("/tenants", tenantCtr.PageList)
	v1RouterGroup.GET("/tenants/:tenantID", tenantCtr.Detail)
	v1RouterGroup.PUT("/tenants/:tenantID", tenantCtr.Update)
	v1RouterGroup.DELETE("/tenants/:tenantID", tenantCtr.Delete)
	// 重置内置管理员密码（R2 动作子路径；builtin-admin 为租户下的单体子资源）
	v1RouterGroup.POST("/tenants/:tenantID/builtin-admin/reset-password", tenantCtr.ResetAdminPassword)
}
