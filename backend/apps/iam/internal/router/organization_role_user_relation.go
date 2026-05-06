package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganizationroleuserrelation"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRoleUserRelationRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserRelationCtr := ctrorganizationroleuserrelation.NewOrganizationRoleUserRelationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/organization/create", organizationRoleUserRelationCtr.Create)
	v1RouterGroup.POST("/organization/delete", organizationRoleUserRelationCtr.Delete)
	v1RouterGroup.POST("/organization/pageList", organizationRoleUserRelationCtr.PageList)
}