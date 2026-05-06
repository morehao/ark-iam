package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrauth"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrconnector"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrsso"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func authRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc("your-jwt-secret-key-change-in-production")
	authCtr := ctrauth.NewAuthCtr(authSvc)
	connectorCtr := ctrconnector.NewConnectorCtr()
	ssoConnectorCtr := ctrsso.NewSsoConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/auth/login", authCtr.Login)
	v1RouterGroup.POST("/auth/register", authCtr.Register)
	v1RouterGroup.POST("/auth/refresh-token", authCtr.RefreshToken)
	v1RouterGroup.POST("/auth/logout", authCtr.Logout)
	v1RouterGroup.GET("/auth/userinfo", authCtr.Userinfo)
	v1RouterGroup.GET("/auth/auth/authorizationUrl", authCtr.GetSsoAuthorizationUrl)
	v1RouterGroup.GET("/auth/auth/callback", authCtr.SsoCallback)

	v1RouterGroup.POST("/auth/connector/create", connectorCtr.Create)
	v1RouterGroup.POST("/auth/connector/delete", connectorCtr.Delete)
	v1RouterGroup.POST("/auth/connector/update", connectorCtr.Update)
	v1RouterGroup.GET("/auth/connector/detail", connectorCtr.Detail)
	v1RouterGroup.POST("/auth/connector/pageList", connectorCtr.PageList)

	v1RouterGroup.POST("/auth/ssoConnector/create", ssoConnectorCtr.Create)
	v1RouterGroup.POST("/auth/ssoConnector/delete", ssoConnectorCtr.Delete)
	v1RouterGroup.POST("/auth/ssoConnector/update", ssoConnectorCtr.Update)
	v1RouterGroup.GET("/auth/ssoConnector/detail", ssoConnectorCtr.Detail)
	v1RouterGroup.POST("/auth/ssoConnector/pageList", ssoConnectorCtr.PageList)
}