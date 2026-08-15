package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func organizationRouter(groups *ginserver.RouterGroups) {
	organizationCtr := ctrtenant.NewOrganizationCtr()
	organizationUserCtr := ctrtenant.NewOrganizationUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	// 组织树
	v1RouterGroup.POST("/organizations", organizationCtr.Create)
	v1RouterGroup.GET("/organizations/tree", organizationCtr.Tree)
	v1RouterGroup.GET("/organizations/:organizationID", organizationCtr.Detail)
	v1RouterGroup.PUT("/organizations/:organizationID", organizationCtr.Update)
	v1RouterGroup.PATCH("/organizations/:organizationID", organizationCtr.UpdateStatus)
	v1RouterGroup.DELETE("/organizations/:organizationID", organizationCtr.Delete)
	// 组织关系
	v1RouterGroup.GET("/organizations/:organizationID/users", organizationUserCtr.PageList)
	v1RouterGroup.POST("/organizations/:organizationID/users", organizationUserCtr.Create)
	v1RouterGroup.PUT("/organizations/:organizationID/users/:userID", organizationUserCtr.Update)
	v1RouterGroup.DELETE("/organizations/:organizationID/users/:userID", organizationUserCtr.Delete)
	v1RouterGroup.GET("/organizations/:organizationID/users/descendants", organizationUserCtr.SubtreeUsers)
	// 用户归属
	v1RouterGroup.GET("/users/:userID/organizations", organizationUserCtr.GetUserOrganizations)
	v1RouterGroup.PUT("/users/:userID/organizations", organizationUserCtr.UpdateUserOrganizations)
}
