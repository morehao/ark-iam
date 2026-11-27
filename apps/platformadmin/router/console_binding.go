package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrconsolebinding"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// consoleBindingRouter Host → 租户/issuer/client 的 binding 管理（平台管理员）。
func consoleBindingRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	ctr := ctrconsolebinding.NewConsoleBindingCtr(db)
	rg.GET("/console-bindings", ctr.List)
	rg.POST("/console-bindings/:bindingID/enable", ctr.Enable)
	rg.POST("/console-bindings/:bindingID/disable", ctr.Disable)
}
