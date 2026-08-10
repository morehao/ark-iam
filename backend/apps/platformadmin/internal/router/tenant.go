package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/tenant/create", tenantCtr.Create)
	v1RouterGroup.POST("/tenant/createAsOwner", tenantCtr.CreateAsOwner)
	v1RouterGroup.POST("/tenant/delete", tenantCtr.Delete)
	v1RouterGroup.POST("/tenant/update", tenantCtr.Update)
	v1RouterGroup.GET("/tenant/detail", tenantCtr.Detail)
	v1RouterGroup.POST("/tenant/pageList", tenantCtr.PageList)
}

func departmentRouter(groups *ginserver.RouterGroups) {
	departmentCtr := ctrtenant.NewDepartmentCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/department/create", departmentCtr.Create)
	v1RouterGroup.POST("/department/delete", departmentCtr.Delete)
	v1RouterGroup.POST("/department/update", departmentCtr.Update)
	v1RouterGroup.GET("/department/detail", departmentCtr.Detail)
	v1RouterGroup.POST("/department/pageList", departmentCtr.PageList)
	v1RouterGroup.GET("/department/tree", departmentCtr.Tree)
}

func systemRouter(groups *ginserver.RouterGroups) {
	systemCtr := ctrtenant.NewSystemCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/system/create", systemCtr.Create)
	v1RouterGroup.POST("/system/delete", systemCtr.Delete)
	v1RouterGroup.POST("/system/update", systemCtr.Update)
	v1RouterGroup.GET("/system/detail", systemCtr.Detail)
	v1RouterGroup.POST("/system/pageList", systemCtr.PageList)
}

func logRouter(groups *ginserver.RouterGroups) {
	logCtr := ctrtenant.NewLogCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/log/detail", logCtr.Detail)
	v1RouterGroup.POST("/log/pageList", logCtr.PageList)
}
