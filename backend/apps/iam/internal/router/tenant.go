package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()
	departmentCtr := ctrtenant.NewDepartmentCtr()
	organizationCtr := ctrtenant.NewOrganizationCtr()
	systemCtr := ctrtenant.NewSystemCtr()
	organizationRoleCtr := ctrtenant.NewOrganizationRoleCtr()
	organizationUserRelationCtr := ctrtenant.NewOrganizationUserRelationCtr()
	organizationRoleUserRelationCtr := ctrtenant.NewOrganizationRoleUserRelationCtr()
	logCtr := ctrtenant.NewLogCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/tenant/create", tenantCtr.Create)
	v1RouterGroup.POST("/tenant/delete", tenantCtr.Delete)
	v1RouterGroup.POST("/tenant/update", tenantCtr.Update)
	v1RouterGroup.GET("/tenant/detail", tenantCtr.Detail)
	v1RouterGroup.POST("/tenant/pageList", tenantCtr.PageList)

	v1RouterGroup.POST("/tenant/department/create", departmentCtr.Create)
	v1RouterGroup.POST("/tenant/department/delete", departmentCtr.Delete)
	v1RouterGroup.POST("/tenant/department/update", departmentCtr.Update)
	v1RouterGroup.GET("/tenant/department/detail", departmentCtr.Detail)
	v1RouterGroup.POST("/tenant/department/pageList", departmentCtr.PageList)
	v1RouterGroup.GET("/tenant/department/tree", departmentCtr.Tree)

	v1RouterGroup.POST("/tenant/organization/create", organizationCtr.Create)
	v1RouterGroup.POST("/tenant/organization/delete", organizationCtr.Delete)
	v1RouterGroup.POST("/tenant/organization/update", organizationCtr.Update)
	v1RouterGroup.GET("/tenant/organization/detail", organizationCtr.Detail)
	v1RouterGroup.POST("/tenant/organization/pageList", organizationCtr.PageList)

	v1RouterGroup.POST("/tenant/system/create", systemCtr.Create)
	v1RouterGroup.POST("/tenant/system/delete", systemCtr.Delete)
	v1RouterGroup.POST("/tenant/system/update", systemCtr.Update)
	v1RouterGroup.GET("/tenant/system/detail", systemCtr.Detail)
	v1RouterGroup.POST("/tenant/system/pageList", systemCtr.PageList)

	v1RouterGroup.POST("/tenant/organization-role/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/tenant/organization-role/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/tenant/organization-role/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/tenant/organization-role/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/tenant/organization-role/pageList", organizationRoleCtr.PageList)

	v1RouterGroup.POST("/tenant/organization-user/create", organizationUserRelationCtr.Create)
	v1RouterGroup.POST("/tenant/organization-user/delete", organizationUserRelationCtr.Delete)
	v1RouterGroup.POST("/tenant/organization-user/pageList", organizationUserRelationCtr.PageList)

	v1RouterGroup.POST("/tenant/organization-role-user/create", organizationRoleUserRelationCtr.Create)
	v1RouterGroup.POST("/tenant/organization-role-user/delete", organizationRoleUserRelationCtr.Delete)
	v1RouterGroup.POST("/tenant/organization-role-user/pageList", organizationRoleUserRelationCtr.PageList)

	v1RouterGroup.GET("/tenant/log/detail", logCtr.Detail)
	v1RouterGroup.POST("/tenant/log/pageList", logCtr.PageList)
}
