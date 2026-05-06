package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganizationrole"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRoleRouter(groups *ginserver.RouterGroups) {
	organizationRoleCtr := ctrorganizationrole.NewOrganizationRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/organizationrole/create", organizationRoleCtr.Create)
	v1RouterGroup.POST("/organizationrole/delete", organizationRoleCtr.Delete)
	v1RouterGroup.POST("/organizationrole/update", organizationRoleCtr.Update)
	v1RouterGroup.GET("/organizationrole/detail", organizationRoleCtr.Detail)
	v1RouterGroup.POST("/organizationrole/pageList", organizationRoleCtr.PageList)
}