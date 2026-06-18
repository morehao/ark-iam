package router

import (
	"crypto/rsa"

	"github.com/morehao/golib/biz/gserver/ginserver"
)

var OIDCPublicKey *rsa.PublicKey

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	tenantRouter(groups)
	apiKeyRouter(groups)
	userRouter(groups)
	roleRouter(groups)
	menuRouter(groups)
	scopeRouter(groups)
	resourceRouter(groups)
	roleMenuRouter(groups)
	roleScopeRouter(groups)
	userRoleRouter(groups)
	oauthClientRouter(groups)
	applicationRouter(groups)
	connectorRouter(groups)
	departmentRouter(groups)
	organizationRouter(groups)
	systemRouter(groups)
	organizationRoleRouter(groups)
	organizationUserRouter(groups)
	organizationRoleUserRouter(groups)
	logRouter(groups)
	domainRouter(groups)
	tenantApplicationRouter(groups)
}
