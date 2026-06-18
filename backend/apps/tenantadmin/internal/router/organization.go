package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrtenant.NewOrganizationRoleCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenantadmin/organizationRole/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/tenantadmin/organizationRole/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/tenantadmin/organizationRole/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/tenantadmin/organizationRole/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/tenantadmin/organizationRole/pageList", organizationRoleCtr.PageList)
}

func organizationUserRouter(groups *ginserver.RouterGroups) {
	organizationUserCtr := ctrtenant.NewOrganizationUserCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenantadmin/organizationUser/create", organizationUserCtr.Create)
	v1RouterGroup.POST("/tenantadmin/organizationUser/delete", organizationUserCtr.Delete)
	v1RouterGroup.POST("/tenantadmin/organizationUser/pageList", organizationUserCtr.PageList)
}

func organizationRoleUserRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserCtr := ctrtenant.NewOrganizationRoleUserCtr()
	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/tenantadmin/organizationRoleUser/create", organizationRoleUserCtr.Create)
	v1RouterGroup.POST("/tenantadmin/organizationRoleUser/delete", organizationRoleUserCtr.Delete)
	v1RouterGroup.POST("/tenantadmin/organizationRoleUser/pageList", organizationRoleUserCtr.PageList)
}
