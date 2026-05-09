package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	tenantRouter(groups)
	userRouter(groups)
	roleRouter(groups)
	menuRouter(groups)
	scopeRouter(groups)
	resourceRouter(groups)
	roleMenuRouter(groups)
	roleScopeRouter(groups)
	userRoleRouter(groups)
	applicationRouter(groups)
	authRouter(groups)
	connectorRouter(groups)
	departmentRouter(groups)
	organizationRouter(groups)
	systemRouter(groups)
	organizationRoleRouter(groups)
	organizationUserRelationRouter(groups)
	organizationRoleUserRelationRouter(groups)
	logRouter(groups)
}
