package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganizationuserrelation"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationUserRelationRouter(groups *ginserver.RouterGroups) {
	organizationUserRelationCtr := ctrorganizationuserrelation.NewOrganizationUserRelationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/organizationuserrelation/create", organizationUserRelationCtr.Create)
	v1RouterGroup.POST("/organizationuserrelation/delete", organizationUserRelationCtr.Delete)
	v1RouterGroup.POST("/organizationuserrelation/pageList", organizationUserRelationCtr.PageList)
}