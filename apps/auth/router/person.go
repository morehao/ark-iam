package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth/controller/ctrperson"
	"github.com/morehao/ark-iam/apps/auth/middleware"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func personRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	personCtr := ctrperson.NewPersonCtr(db)
	rg.GET("/me", personCtr.Detail)
	// 改密码按 IP 限流，防止对旧密码的暴力试探（H7）
	rg.POST("/me/changePassword", middleware.PasswordChangeRateLimit(), personCtr.UpdatePassword)
}
