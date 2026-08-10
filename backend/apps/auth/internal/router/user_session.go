package router

import (
	"github.com/morehao/ark-iam/auth/internal/controller/ctrsession"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userSessionRouter(groups *ginserver.RouterGroups) {
	sessionCtr := ctrsession.NewSessionCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/user/sessions", sessionCtr.List)
	v1RouterGroup.DELETE("/user/sessions", sessionCtr.RevokeAll)
	v1RouterGroup.DELETE("/user/sessions/:sessionId", sessionCtr.Revoke)
}
