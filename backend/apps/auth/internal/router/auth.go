package router

import (
	"github.com/morehao/ark-iam/auth/internal/controller/ctrauth"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func authRouter(groups *ginserver.RouterGroups) {
	authSvc := svcauth.NewAuthSvc()
	authCtr := ctrauth.NewAuthCtr(authSvc)

	// 认证相关操作直接挂在服务标识段下（/v1/auth/{operation}），
	// 避免出现 /v1/auth/auth/register 的冗余路径。
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/myTenants", authCtr.MyTenants)
	v1RouterGroup.POST("/register", authCtr.Register)
	v1RouterGroup.POST("/joinTenant", authCtr.JoinTenant)
	v1RouterGroup.POST("/logout", authCtr.Logout)
	v1RouterGroup.POST("/logoutAll", authCtr.LogoutAll)
	v1RouterGroup.GET("/userinfo", authCtr.Userinfo)
}

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
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
