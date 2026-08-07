package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/tenant/create", tenantCtr.Create)
	v1RouterGroup.POST("/platformadmin/tenant/delete", tenantCtr.Delete)
	v1RouterGroup.POST("/platformadmin/tenant/update", tenantCtr.Update)
	v1RouterGroup.GET("/platformadmin/tenant/detail", tenantCtr.Detail)
	v1RouterGroup.POST("/platformadmin/tenant/pageList", tenantCtr.PageList)
}

func departmentRouter(groups *ginserver.RouterGroups) {
	departmentCtr := ctrtenant.NewDepartmentCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/department/create", departmentCtr.Create)
	v1RouterGroup.POST("/platformadmin/department/delete", departmentCtr.Delete)
	v1RouterGroup.POST("/platformadmin/department/update", departmentCtr.Update)
	v1RouterGroup.GET("/platformadmin/department/detail", departmentCtr.Detail)
	v1RouterGroup.POST("/platformadmin/department/pageList", departmentCtr.PageList)
	v1RouterGroup.GET("/platformadmin/department/tree", departmentCtr.Tree)
}

func organizationRouter(groups *ginserver.RouterGroups) {
	organizationCtr := ctrtenant.NewOrganizationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/organization/create", organizationCtr.Create)
	v1RouterGroup.POST("/platformadmin/organization/delete", organizationCtr.Delete)
	v1RouterGroup.POST("/platformadmin/organization/update", organizationCtr.Update)
	v1RouterGroup.GET("/platformadmin/organization/detail", organizationCtr.Detail)
	v1RouterGroup.POST("/platformadmin/organization/pageList", organizationCtr.PageList)
}

func systemRouter(groups *ginserver.RouterGroups) {
	systemCtr := ctrtenant.NewSystemCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/system/create", systemCtr.Create)
	v1RouterGroup.POST("/platformadmin/system/delete", systemCtr.Delete)
	v1RouterGroup.POST("/platformadmin/system/update", systemCtr.Update)
	v1RouterGroup.GET("/platformadmin/system/detail", systemCtr.Detail)
	v1RouterGroup.POST("/platformadmin/system/pageList", systemCtr.PageList)
}

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrtenant.NewOrganizationRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/organizationRole/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/platformadmin/organizationRole/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/platformadmin/organizationRole/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/platformadmin/organizationRole/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/platformadmin/organizationRole/pageList", organizationRoleCtr.PageList)
}

func organizationUserRouter(groups *ginserver.RouterGroups) {
	organizationUserCtr := ctrtenant.NewOrganizationUserCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/organizationUser/create", organizationUserCtr.Create)
	v1RouterGroup.POST("/platformadmin/organizationUser/delete", organizationUserCtr.Delete)
	v1RouterGroup.POST("/platformadmin/organizationUser/pageList", organizationUserCtr.PageList)
}

func organizationRoleUserRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserCtr := ctrtenant.NewOrganizationRoleUserCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/organizationRoleUser/create", organizationRoleUserCtr.Create)
	v1RouterGroup.POST("/platformadmin/organizationRoleUser/delete", organizationRoleUserCtr.Delete)
	v1RouterGroup.POST("/platformadmin/organizationRoleUser/pageList", organizationRoleUserCtr.PageList)
}

func logRouter(groups *ginserver.RouterGroups) {
	logCtr := ctrtenant.NewLogCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/platformadmin/log/detail", logCtr.Detail)
	v1RouterGroup.POST("/platformadmin/log/pageList", logCtr.PageList)
}
