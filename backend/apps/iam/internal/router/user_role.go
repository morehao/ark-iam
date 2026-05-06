package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctruserrole"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRoleRouter(groups *ginserver.RouterGroups) {
	userRoleCtr := ctruserrole.NewUserRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/userrole/create", userRoleCtr.Create)
	v1RouterGroup.POST("/userrole/delete", userRoleCtr.Delete)
	v1RouterGroup.POST("/userrole/pageList", userRoleCtr.PageList)
}