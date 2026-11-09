package router

import (
	"github.com/morehao/ark-iam/iam/tenantadmin/controller/ctrtenant"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func userRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	userCtr := ctrtenant.NewUserCtr(db)

	rg.GET("/users", userCtr.PageList)
	rg.POST("/users", userCtr.Create)
	rg.GET("/users/:userID", userCtr.Detail)
	rg.PATCH("/users/:userID", userCtr.Update)
	rg.POST("/users/:userID/reset-password", userCtr.ResetPassword)
	// 用户角色（用户侧授权入口）
	rg.GET("/users/:userID/roles", userCtr.ListRoles)
	rg.PUT("/users/:userID/roles", userCtr.UpdateRoles)
	// 用户应用准入（准入层唯一写入口：无角色的应用同样在此授权）
	rg.GET("/users/:userID/applications", userCtr.ListApplications)
	rg.PUT("/users/:userID/applications", userCtr.UpdateApplications)
	// 用户第三方身份（用户视角子资源）
	rg.GET("/users/:userID/identities", userCtr.ListIdentities)
	rg.POST("/users/:userID/identities", userCtr.CreateIdentity)
	rg.DELETE("/users/:userID/identities/:identityID", userCtr.DeleteIdentity)
	// 用户登录日志（用户视角子资源）
	rg.GET("/users/:userID/login-logs", userCtr.ListLoginLogs)
}
