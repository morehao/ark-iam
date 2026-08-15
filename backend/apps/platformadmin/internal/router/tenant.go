package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()
	organizationCtr := ctrtenant.NewOrganizationCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/tenants", tenantCtr.Create)
	v1RouterGroup.GET("/tenants", tenantCtr.PageList)
	v1RouterGroup.GET("/tenants/:tenantID", tenantCtr.Detail)
	v1RouterGroup.PUT("/tenants/:tenantID", tenantCtr.Update)
	v1RouterGroup.DELETE("/tenants/:tenantID", tenantCtr.Delete)
	v1RouterGroup.POST("/tenants/createAsOwner", tenantCtr.CreateAsOwner)
	// 组织只读（跨租户排查）
	v1RouterGroup.GET("/organizations/tree", organizationCtr.Tree)
	v1RouterGroup.GET("/users/:userID/organizations", organizationCtr.GetUserOrganizations)
}

func systemRouter(groups *ginserver.RouterGroups) {
	systemCtr := ctrtenant.NewSystemCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/systems", systemCtr.Create)
	v1RouterGroup.GET("/systems", systemCtr.PageList)
	v1RouterGroup.GET("/systems/:systemID", systemCtr.Detail)
	v1RouterGroup.PUT("/systems/:systemID", systemCtr.Update)
	v1RouterGroup.DELETE("/systems/:systemID", systemCtr.Delete)
}

func logRouter(groups *ginserver.RouterGroups) {
	logCtr := ctrtenant.NewLogCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/logs", logCtr.PageList)
	v1RouterGroup.GET("/logs/:logID", logCtr.Detail)
}
