package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenant/create", tenantCtr.Create)
	v1RouterGroup.POST("/tenant/delete", tenantCtr.Delete)
	v1RouterGroup.POST("/tenant/update", tenantCtr.Update)
	v1RouterGroup.GET("/tenant/detail", tenantCtr.Detail)
	v1RouterGroup.POST("/tenant/pageList", tenantCtr.PageList)
}

func departmentRouter(groups *ginserver.RouterGroups) {
	departmentCtr := ctrtenant.NewDepartmentCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/department/create", departmentCtr.Create)
	v1RouterGroup.POST("/department/delete", departmentCtr.Delete)
	v1RouterGroup.POST("/department/update", departmentCtr.Update)
	v1RouterGroup.GET("/department/detail", departmentCtr.Detail)
	v1RouterGroup.POST("/department/pageList", departmentCtr.PageList)
	v1RouterGroup.GET("/department/tree", departmentCtr.Tree)
}

func organizationRouter(groups *ginserver.RouterGroups) {
	organizationCtr := ctrtenant.NewOrganizationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/organization/create", organizationCtr.Create)
	v1RouterGroup.POST("/organization/delete", organizationCtr.Delete)
	v1RouterGroup.POST("/organization/update", organizationCtr.Update)
	v1RouterGroup.GET("/organization/detail", organizationCtr.Detail)
	v1RouterGroup.POST("/organization/pageList", organizationCtr.PageList)
}

func systemRouter(groups *ginserver.RouterGroups) {
	systemCtr := ctrtenant.NewSystemCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/system/create", systemCtr.Create)
	v1RouterGroup.POST("/system/delete", systemCtr.Delete)
	v1RouterGroup.POST("/system/update", systemCtr.Update)
	v1RouterGroup.GET("/system/detail", systemCtr.Detail)
	v1RouterGroup.POST("/system/pageList", systemCtr.PageList)
}

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrtenant.NewOrganizationRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/organizationRole/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/organizationRole/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/organizationRole/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/organizationRole/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/organizationRole/pageList", organizationRoleCtr.PageList)
}

func organizationUserRelationRouter(groups *ginserver.RouterGroups) {
	organizationUserRelationCtr := ctrtenant.NewOrganizationUserRelationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/organizationUser/create", organizationUserRelationCtr.Create)
	v1RouterGroup.POST("/organizationUser/delete", organizationUserRelationCtr.Delete)
	v1RouterGroup.POST("/organizationUser/pageList", organizationUserRelationCtr.PageList)
}

func organizationRoleUserRelationRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserRelationCtr := ctrtenant.NewOrganizationRoleUserRelationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/organizationRoleUser/create", organizationRoleUserRelationCtr.Create)
	v1RouterGroup.POST("/organizationRoleUser/delete", organizationRoleUserRelationCtr.Delete)
	v1RouterGroup.POST("/organizationRoleUser/pageList", organizationRoleUserRelationCtr.PageList)
}

func logRouter(groups *ginserver.RouterGroups) {
	logCtr := ctrtenant.NewLogCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/log/detail", logCtr.Detail)
	v1RouterGroup.POST("/log/pageList", logCtr.PageList)
}
