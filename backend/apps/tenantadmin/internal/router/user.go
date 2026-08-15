package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctrtenant.NewUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/users", userCtr.PageList)
	v1RouterGroup.POST("/users", userCtr.Create)
	v1RouterGroup.GET("/users/:userID", userCtr.Detail)
	v1RouterGroup.PATCH("/users/:userID", userCtr.Update)
	v1RouterGroup.POST("/users/:userID/reset-password", userCtr.ResetPassword)
	// 用户角色（用户侧授权入口）
	v1RouterGroup.GET("/users/:userID/roles", userCtr.ListRoles)
	v1RouterGroup.PUT("/users/:userID/roles", userCtr.UpdateRoles)
}
