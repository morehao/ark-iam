package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups) {
	userRouter(groups)
	machineUserRouter(groups)
	departmentRouter(groups)
	roleRouter(groups)
	tenantMenuRouter(groups)
	apiKeyRouter(groups)
	inviteRouter(groups)
}
