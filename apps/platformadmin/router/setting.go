package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/controller/ctrsetting"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func settingRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	settingCtr := ctrsetting.NewSettingCtr(db)
	rg.POST("/settings", settingCtr.Create)
	rg.GET("/settings", settingCtr.PageList)
	// 注意：/settings/groups 必须注册在 /settings/:settingID 之前，否则 groups 会被 :settingID 匹配。
	rg.GET("/settings/groups", settingCtr.Groups)
	rg.GET("/settings/:settingID", settingCtr.Detail)
	rg.PUT("/settings/:settingID", settingCtr.Update)
	rg.PATCH("/settings/:settingID/status", settingCtr.UpdateStatus)
	rg.DELETE("/settings/:settingID", settingCtr.Delete)
}
