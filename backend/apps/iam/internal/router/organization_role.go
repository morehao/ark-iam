package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganizationrole"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrorganizationrole.NewOrganizationRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/organization/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/organization/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/organization/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/organization/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/organization/pageList", organizationRoleCtr.PageList)
}