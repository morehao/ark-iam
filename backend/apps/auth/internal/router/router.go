package router

import (
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// RegisterRouter registers all auth application routes.
// OIDC and docs setup happen in InitOIDC / the app init path, not here.
func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	authRouter(groups)
	personRouter(groups)
	userSessionRouter(groups)
	connectorRouter(groups)
}
