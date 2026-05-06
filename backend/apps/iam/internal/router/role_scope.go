package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrrole_scope"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func roleScopeRouter(groups *ginserver.RouterGroups) {
	roleScopeCtr := ctrrole_scope.NewRoleScopeCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/roleScope/create", roleScopeCtr.Create)
	v1RouterGroup.POST("/roleScope/delete", roleScopeCtr.Delete)
	v1RouterGroup.POST("/roleScope/pageList", roleScopeCtr.PageList)
}