package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups) {
	tenantRouter(groups)
	systemRouter(groups)
	logRouter(groups)
	apiKeyRouter(groups)
	userRouter(groups)
	roleRouter(groups)
	menuRouter(groups)
	applicationClientRouter(groups)
	applicationRouter(groups)
	domainRouter(groups)
	tenantApplicationRouter(groups)
}
