package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	tenantRouter(groups)
	systemRouter(groups)
	logRouter(groups)
	departmentRouter(groups)
	menuRouter(groups)
	connectorRouter(groups)
	ssoConnectorRouter(groups)
	userRouter(groups)
	roleRouter(groups)
	applicationRouter(groups)
	resourceRouter(groups)
	scopeRouter(groups)
	roleScopeRouter(groups)
	userRoleRouter(groups)
	roleMenuRouter(groups)
	organizationRouter(groups)
	organizationRoleRouter(groups)
	organizationUserRelationRouter(groups)
	organizationRoleUserRelationRouter(groups)
}
