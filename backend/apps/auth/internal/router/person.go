package router

import (
	"github.com/morehao/ark-iam/auth/internal/controller/ctrperson"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func personRouter(groups *ginserver.RouterGroups) {
	personCtr := ctrperson.NewPersonCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/me", personCtr.Detail)
	// 改密码按 IP 限流，防止对旧密码的暴力试探（H7）
	v1RouterGroup.POST("/me/changePassword", middleware.PasswordChangeRateLimit(), personCtr.UpdatePassword)
}
