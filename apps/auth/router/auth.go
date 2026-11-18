package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth/controller/ctrauth"
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

// connectorRouter 连接器「运行时」路由（登录链路）。
// 管理面（CRUD/工厂/测试）归平台运营端 /v1/platform/connectors*（platformadmin）。
func connectorRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	connectorCtr := ctrauth.NewConnectorCtr(db)
	rg.POST("/connectors/:connectorID/authorize", connectorCtr.Authorize)
	rg.GET("/connectors/callback", connectorCtr.Callback)
}
