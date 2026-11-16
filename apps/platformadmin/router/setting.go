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
	rg.GET("/settings/:settingID", settingCtr.Detail)
	rg.PUT("/settings/:settingID", settingCtr.Update)
	rg.PATCH("/settings/:settingID/status", settingCtr.UpdateStatus)
	rg.DELETE("/settings/:settingID", settingCtr.Delete)
}
