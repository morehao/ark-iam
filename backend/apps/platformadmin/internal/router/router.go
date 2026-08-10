package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	tenantRouter(groups)
	departmentRouter(groups)
	systemRouter(groups)
	logRouter(groups)
	apiKeyRouter(groups)
	userRouter(groups)
	roleRouter(groups)
	menuRouter(groups)
	scopeRouter(groups)
	resourceRouter(groups)
	roleMenuRouter(groups)
	roleScopeRouter(groups)
	userRoleRouter(groups)
	applicationClientRouter(groups)
	applicationRouter(groups)
	domainRouter(groups)
	tenantApplicationRouter(groups)
}
