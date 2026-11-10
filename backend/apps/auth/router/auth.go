package router

import (
	"github.com/morehao/ark-iam/auth/controller/ctrauth"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func authRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	authCtr := ctrauth.NewAuthCtr(db)
	// 认证相关操作直接挂在服务标识段下（/v1/auth/{operation}），
	// 避免出现 /v1/auth/auth/register 的冗余路径。
	rg.GET("/me/tenants", authCtr.MyTenants)
	rg.POST("/joinTenant", authCtr.JoinTenant)
	rg.POST("/logout", authCtr.Logout)
	rg.POST("/logoutAll", authCtr.LogoutAll)
	rg.GET("/userinfo", authCtr.Userinfo)
}

func connectorRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	connectorCtr := ctrauth.NewConnectorCtr(db)
	rg.POST("/connectors", connectorCtr.Create)
	rg.GET("/connectors", connectorCtr.PageList)
	rg.GET("/connector-factories", connectorCtr.GetFactoryList)
	rg.GET("/connectors/:connectorID", connectorCtr.Detail)
	rg.PUT("/connectors/:connectorID", connectorCtr.Update)
	rg.DELETE("/connectors/:connectorID", connectorCtr.Delete)
	rg.POST("/connectors/:connectorID/test", connectorCtr.TestConnector)
	rg.POST("/connectors/:connectorID/authorize", connectorCtr.Authorize)
	rg.GET("/connectors/callback", connectorCtr.Callback)
}
