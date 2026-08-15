package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctruser"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// userRouter 平台排查视角：跨租户用户目录只读 + 挂起/恢复 + 重置密码；
// 用户创建/编辑/删除与租户内组织归属/角色分配收敛到 tenantadmin。
func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctruser.NewUserCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)

	// 用户资源
	v1RouterGroup.GET("/users", userCtr.PageList)
	v1RouterGroup.GET("/users/:userID", userCtr.Detail)
	v1RouterGroup.PATCH("/users/:userID", userCtr.UpdateStatus)
	v1RouterGroup.POST("/users/:userID/changePassword", userCtr.UpdatePassword)

	// 用户身份（用户视角子资源）
	v1RouterGroup.GET("/users/:userID/identities", userCtr.GetUserIdentityByUser)
	v1RouterGroup.POST("/users/:userID/identities", userCtr.CreateUserIdentity)
	v1RouterGroup.DELETE("/users/:userID/identities/:identityID", userCtr.DeleteUserIdentity)

	// 用户登录日志（用户视角子资源）
	v1RouterGroup.GET("/users/:userID/login-logs", userCtr.GetUserLoginLogByUser)
}
