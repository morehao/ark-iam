package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganization_role"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrorganization_role.NewOrganizationRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/organizationRole/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/organizationRole/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/organizationRole/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/organizationRole/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/organizationRole/pageList", organizationRoleCtr.PageList)
}