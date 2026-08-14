package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	organizationRouter(groups)
	organizationRoleRouter(groups)
	organizationUserRouter(groups)
	organizationRoleUserRouter(groups)
	tenantMenuRouter(groups)
}
