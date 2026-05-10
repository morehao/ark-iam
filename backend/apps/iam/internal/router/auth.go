package router

import (
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrauth"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func authRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc(config.Conf.JWT.SignKey)
	authCtr := ctrauth.NewAuthCtr(authSvc)

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/login", authCtr.Login)
	v1RouterGroup.GET("/myTenants", authCtr.MyTenants)
	v1RouterGroup.POST("/selectTenant", authCtr.SelectTenant)
	v1RouterGroup.POST("/switchTenant", authCtr.SwitchTenant)
	v1RouterGroup.POST("/register", authCtr.Register)
	v1RouterGroup.POST("/refreshToken", authCtr.RefreshToken)
	v1RouterGroup.POST("/logout", authCtr.Logout)
	v1RouterGroup.POST("/logoutAll", authCtr.LogoutAll)
	v1RouterGroup.GET("/userinfo", authCtr.Userinfo)
}

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/connector/create", connectorCtr.Create)
	v1RouterGroup.POST("/connector/delete", connectorCtr.Delete)
	v1RouterGroup.POST("/connector/update", connectorCtr.Update)
	v1RouterGroup.GET("/connector/detail", connectorCtr.Detail)
	v1RouterGroup.POST("/connector/pageList", connectorCtr.PageList)
	v1RouterGroup.POST("/connector/getFactoryList", connectorCtr.GetFactoryList)
	v1RouterGroup.POST("/connector/:connectorId/test", connectorCtr.TestConnector)
	v1RouterGroup.POST("/connector/:connectorId/authorize", connectorCtr.Authorize)
	v1RouterGroup.GET("/connector/callback", connectorCtr.Callback)
}
