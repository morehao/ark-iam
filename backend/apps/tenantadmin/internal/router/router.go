package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups) {
	userRouter(groups)
	organizationRouter(groups)
	roleRouter(groups)
	tenantMenuRouter(groups)
	inviteRouter(groups)
}
