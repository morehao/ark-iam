package router

import (
	"github.com/morehao/ark-iam/auth/internal/controller/ctrauth"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func authRouter(groups *ginserver.RouterGroups) {
	authCtr := ctrauth.NewAuthCtr()
	// 认证相关操作直接挂在服务标识段下（/v1/auth/{operation}），
	// 避免出现 /v1/auth/auth/register 的冗余路径。
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/me/tenants", authCtr.MyTenants)
	v1RouterGroup.POST("/register", authCtr.Register)
	v1RouterGroup.POST("/joinTenant", authCtr.JoinTenant)
	v1RouterGroup.POST("/logout", authCtr.Logout)
	v1RouterGroup.POST("/logoutAll", authCtr.LogoutAll)
	v1RouterGroup.GET("/userinfo", authCtr.Userinfo)
}

func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/connectors", connectorCtr.Create)
	v1RouterGroup.GET("/connectors", connectorCtr.PageList)
	v1RouterGroup.GET("/connector-factories", connectorCtr.GetFactoryList)
	v1RouterGroup.GET("/connectors/:connectorID", connectorCtr.Detail)
	v1RouterGroup.PUT("/connectors/:connectorID", connectorCtr.Update)
	v1RouterGroup.DELETE("/connectors/:connectorID", connectorCtr.Delete)
	v1RouterGroup.POST("/connectors/:connectorID/test", connectorCtr.TestConnector)
	v1RouterGroup.POST("/connectors/:connectorID/authorize", connectorCtr.Authorize)
	v1RouterGroup.GET("/connectors/callback", connectorCtr.Callback)
}
