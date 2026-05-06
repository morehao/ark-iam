package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganization_user_relation"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationUserRelationRouter(groups *ginserver.RouterGroups) {
	organizationUserRelationCtr := ctrorganization_user_relation.NewOrganizationUserRelationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/organizationUserRelation/create", organizationUserRelationCtr.Create)
	v1RouterGroup.POST("/organizationUserRelation/delete", organizationUserRelationCtr.Delete)
	v1RouterGroup.POST("/organizationUserRelation/pageList", organizationUserRelationCtr.PageList)
}