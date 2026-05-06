package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctruser_role"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRoleRouter(groups *ginserver.RouterGroups) {
	userRoleCtr := ctruser_role.NewUserRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/userRole/create", userRoleCtr.Create)
	v1RouterGroup.POST("/userRole/delete", userRoleCtr.Delete)
	v1RouterGroup.POST("/userRole/pageList", userRoleCtr.PageList)
}