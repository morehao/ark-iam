package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganization_role_user_relation"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRoleUserRelationRouter(groups *ginserver.RouterGroups) {
	organizationRoleUserRelationCtr := ctrorganization_role_user_relation.NewOrganizationRoleUserRelationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/organizationRoleUserRelation/create", organizationRoleUserRelationCtr.Create)
	v1RouterGroup.POST("/organizationRoleUserRelation/delete", organizationRoleUserRelationCtr.Delete)
	v1RouterGroup.POST("/organizationRoleUserRelation/pageList", organizationRoleUserRelationCtr.PageList)
}