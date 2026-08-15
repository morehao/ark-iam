package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctruser"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctruser.NewUserCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)

	// 用户资源
	v1RouterGroup.POST("/users", userCtr.Create)
	v1RouterGroup.GET("/users", userCtr.PageList)
	v1RouterGroup.GET("/users/:userID", userCtr.Detail)
	v1RouterGroup.PUT("/users/:userID", userCtr.Update)
	v1RouterGroup.PATCH("/users/:userID", userCtr.UpdateStatus)
	v1RouterGroup.DELETE("/users/:userID", userCtr.Delete)
	v1RouterGroup.POST("/users/:userID/changePassword", userCtr.UpdatePassword)

	// 用户身份（用户视角子资源 + 顶层全局检索）
	v1RouterGroup.GET("/user-identities", userCtr.PageListUserIdentity)
	v1RouterGroup.GET("/users/:userID/identities", userCtr.GetUserIdentityByUser)
	v1RouterGroup.POST("/users/:userID/identities", userCtr.CreateUserIdentity)
	v1RouterGroup.GET("/users/:userID/identities/:identityID", userCtr.DetailUserIdentity)
	v1RouterGroup.PUT("/users/:userID/identities/:identityID", userCtr.UpdateUserIdentity)
	v1RouterGroup.DELETE("/users/:userID/identities/:identityID", userCtr.DeleteUserIdentity)

	// 用户登录日志（用户视角子资源 + 顶层全局检索）
	v1RouterGroup.GET("/login-logs", userCtr.PageListUserLoginLog)
	v1RouterGroup.GET("/login-logs/:loginLogID", userCtr.DetailUserLoginLog)
	v1RouterGroup.GET("/users/:userID/login-logs", userCtr.GetUserLoginLogByUser)

	// 用户部门关联
}
