package router

import (
	"github.com/morehao/ark-iam/auth/internal/controller/ctrsession"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userSessionRouter(groups *ginserver.RouterGroups) {
	sessionCtr := ctrsession.NewSessionCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/me/sessions", sessionCtr.List)
	v1RouterGroup.DELETE("/me/sessions", sessionCtr.RevokeAll)
	v1RouterGroup.DELETE("/me/sessions/:sessionID", sessionCtr.Revoke)
}
