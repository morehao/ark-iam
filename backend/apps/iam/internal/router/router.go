package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	tenantRouter(groups)
	userRouter(groups)
	permissionRouter(groups)
	authRouter(groups)
}