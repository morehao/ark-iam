package router

import (
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/controller/ctrauth"
	"github.com/morehao/ark-iam/auth/internal/controller/ctrsession"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func authRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc(config.Conf.JWT.SignKey)
	authCtr := ctrauth.NewAuthCtr(authSvc)
	sessionCtr := ctrsession.NewSessionCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/auth/login", authCtr.Login)
	v1RouterGroup.GET("/auth/myTenants", authCtr.MyTenants)
	v1RouterGroup.POST("/auth/selectTenant", authCtr.SelectTenant)
	v1RouterGroup.POST("/auth/switchTenant", authCtr.SwitchTenant)
	v1RouterGroup.POST("/auth/register", authCtr.Register)
	v1RouterGroup.POST("/auth/joinTenant", authCtr.JoinTenant)
	v1RouterGroup.POST("/auth/refreshToken", authCtr.RefreshToken)
	v1RouterGroup.POST("/auth/logout", authCtr.Logout)
	v1RouterGroup.POST("/auth/logoutAll", authCtr.LogoutAll)
	v1RouterGroup.GET("/auth/userinfo", authCtr.Userinfo)

	v1RouterGroup.GET("/auth/user/sessions", sessionCtr.List)
	v1RouterGroup.DELETE("/auth/user/sessions", sessionCtr.RevokeAll)
	v1RouterGroup.DELETE("/auth/user/sessions/:sessionId", sessionCtr.Revoke)
}

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/auth/connector/callback", connectorCtr.Callback)
}
