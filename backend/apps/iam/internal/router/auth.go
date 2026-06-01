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
