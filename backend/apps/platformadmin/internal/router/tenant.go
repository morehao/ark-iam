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
	v1RouterGroup.POST("/tenants/createAsOwner", tenantCtr.CreateAsOwner)
}

func departmentRouter(groups *ginserver.RouterGroups) {
	departmentCtr := ctrtenant.NewDepartmentCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/departments", departmentCtr.Create)
	v1RouterGroup.GET("/departments", departmentCtr.PageList)
	v1RouterGroup.GET("/departments/tree", departmentCtr.Tree)
	v1RouterGroup.GET("/departments/:departmentID", departmentCtr.Detail)
	v1RouterGroup.PUT("/departments/:departmentID", departmentCtr.Update)
	v1RouterGroup.DELETE("/departments/:departmentID", departmentCtr.Delete)
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
