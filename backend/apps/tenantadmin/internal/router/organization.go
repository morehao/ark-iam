package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRouter(groups *ginserver.RouterGroups) {
	organizationCtr := ctrtenant.NewOrganizationCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/organizations", organizationCtr.Create)
	v1RouterGroup.GET("/organizations", organizationCtr.PageList)
	v1RouterGroup.GET("/organizations/:organizationID", organizationCtr.Detail)
	v1RouterGroup.PUT("/organizations/:organizationID", organizationCtr.Update)
	v1RouterGroup.DELETE("/organizations/:organizationID", organizationCtr.Delete)
}

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrtenant.NewOrganizationRoleCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/organization-roles", organizationRoleCtr.Create)
	v1RouterGroup.GET("/organization-roles", organizationRoleCtr.PageList)
	v1RouterGroup.GET("/organization-roles/:organizationRoleID", organizationRoleCtr.Detail)
	v1RouterGroup.PUT("/organization-roles/:organizationRoleID", organizationRoleCtr.Update)
	v1RouterGroup.DELETE("/organization-roles/:organizationRoleID", organizationRoleCtr.Delete)
}

func organizationUserRouter(groups *ginserver.RouterGroups) {
	organizationUserCtr := ctrtenant.NewOrganizationUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/organization-users", organizationUserCtr.PageList)
	v1RouterGroup.POST("/organization-users", organizationUserCtr.Create)
	v1RouterGroup.DELETE("/organization-users/:organizationID/:userID", organizationUserCtr.Delete)
}

func organizationRoleUserRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserCtr := ctrtenant.NewOrganizationRoleUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/organization-role-users", organizationRoleUserCtr.PageList)
	v1RouterGroup.POST("/organization-role-users", organizationRoleUserCtr.Create)
	v1RouterGroup.DELETE("/organization-role-users/:organizationRoleID/:userID", organizationRoleUserCtr.Delete)
}
