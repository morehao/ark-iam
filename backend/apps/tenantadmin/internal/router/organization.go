package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRouter(groups *ginserver.RouterGroups) {
	organizationCtr := ctrtenant.NewOrganizationCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/organization/create", organizationCtr.Create)
	v1RouterGroup.POST("/organization/delete", organizationCtr.Delete)
	v1RouterGroup.POST("/organization/update", organizationCtr.Update)
	v1RouterGroup.GET("/organization/detail", organizationCtr.Detail)
	v1RouterGroup.POST("/organization/pageList", organizationCtr.PageList)
}

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrtenant.NewOrganizationRoleCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/organizationRole/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/organizationRole/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/organizationRole/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/organizationRole/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/organizationRole/pageList", organizationRoleCtr.PageList)
}

func organizationUserRouter(groups *ginserver.RouterGroups) {
	organizationUserCtr := ctrtenant.NewOrganizationUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/organizationUser/create", organizationUserCtr.Create)
	v1RouterGroup.POST("/organizationUser/delete", organizationUserCtr.Delete)
	v1RouterGroup.POST("/organizationUser/pageList", organizationUserCtr.PageList)
}

func organizationRoleUserRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserCtr := ctrtenant.NewOrganizationRoleUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/organizationRoleUser/create", organizationRoleUserCtr.Create)
	v1RouterGroup.POST("/organizationRoleUser/delete", organizationRoleUserCtr.Delete)
	v1RouterGroup.POST("/organizationRoleUser/pageList", organizationRoleUserCtr.PageList)
}
