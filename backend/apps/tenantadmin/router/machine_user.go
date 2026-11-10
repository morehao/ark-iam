package router

import (
	"github.com/morehao/ark-iam/tenantadmin/controller/ctrtenant"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func machineUserRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	machineUserCtr := ctrtenant.NewMachineUserCtr(db)

	rg.POST("/machine-users", machineUserCtr.Create)
	rg.GET("/machine-users", machineUserCtr.PageList)
	rg.GET("/machine-users/:machineUserID", machineUserCtr.Detail)
	rg.PUT("/machine-users/:machineUserID", machineUserCtr.Update)
	rg.PATCH("/machine-users/:machineUserID", machineUserCtr.UpdateStatus)
	rg.DELETE("/machine-users/:machineUserID", machineUserCtr.Delete)
	// 服务账号角色（服务账号侧授权入口）
	rg.GET("/machine-users/:machineUserID/roles", machineUserCtr.ListRoles)
	rg.PUT("/machine-users/:machineUserID/roles", machineUserCtr.UpdateRoles)
}
