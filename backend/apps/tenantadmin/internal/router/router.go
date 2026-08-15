package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	userRouter(groups)
	organizationRouter(groups)
	tenantMenuRouter(groups)
}
