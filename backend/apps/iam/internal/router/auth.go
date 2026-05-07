package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrauth"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func authRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc("your-jwt-secret-key-change-in-production")
	authCtr := ctrauth.NewAuthCtr(authSvc)

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/login", authCtr.Login)
	v1RouterGroup.POST("/register", authCtr.Register)
	v1RouterGroup.POST("/refreshToken", authCtr.RefreshToken)
	v1RouterGroup.POST("/logout", authCtr.Logout)
	v1RouterGroup.GET("/userinfo", authCtr.Userinfo)
	v1RouterGroup.GET("/authorizationUrl", authCtr.GetSsoAuthorizationUrl)
	v1RouterGroup.GET("/callback", authCtr.SsoCallback)
}

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/connector/create", connectorCtr.Create)
	v1RouterGroup.POST("/connector/delete", connectorCtr.Delete)
	v1RouterGroup.POST("/connector/update", connectorCtr.Update)
	v1RouterGroup.GET("/connector/detail", connectorCtr.Detail)
	v1RouterGroup.POST("/connector/pageList", connectorCtr.PageList)
}

func ssoConnectorRouter(groups *ginserver.RouterGroups) {
	ssoConnectorCtr := ctrauth.NewSsoConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/ssoConnector/create", ssoConnectorCtr.Create)
	v1RouterGroup.POST("/ssoConnector/delete", ssoConnectorCtr.Delete)
	v1RouterGroup.POST("/ssoConnector/update", ssoConnectorCtr.Update)
	v1RouterGroup.GET("/ssoConnector/detail", ssoConnectorCtr.Detail)
	v1RouterGroup.POST("/ssoConnector/pageList", ssoConnectorCtr.PageList)
}