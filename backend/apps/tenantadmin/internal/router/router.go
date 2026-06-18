package router

import (
	"crypto/rsa"

	"github.com/morehao/golib/biz/gserver/ginserver"
)

var OIDCPublicKey *rsa.PublicKey

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	organizationRoleRouter(groups)
	organizationUserRouter(groups)
	organizationRoleUserRouter(groups)
}
