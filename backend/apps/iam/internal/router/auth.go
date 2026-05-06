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

	v1RouterGroup.POST("/auth/login", authCtr.Login)
	v1RouterGroup.POST("/auth/register", authCtr.Register)
	v1RouterGroup.POST("/auth/refresh-token", authCtr.RefreshToken)
	v1RouterGroup.POST("/auth/logout", authCtr.Logout)
	v1RouterGroup.GET("/auth/userinfo", authCtr.Userinfo)
	v1RouterGroup.GET("/auth/sso/authorizationUrl", authCtr.GetSsoAuthorizationUrl)
	v1RouterGroup.GET("/auth/sso/callback", authCtr.SsoCallback)
}