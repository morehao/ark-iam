package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func machineUserRouter(groups *ginserver.RouterGroups) {
	machineUserCtr := ctrtenant.NewMachineUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/machine-users", machineUserCtr.Create)
	v1RouterGroup.GET("/machine-users", machineUserCtr.PageList)
	v1RouterGroup.GET("/machine-users/:machineUserID", machineUserCtr.Detail)
	v1RouterGroup.PUT("/machine-users/:machineUserID", machineUserCtr.Update)
	v1RouterGroup.PATCH("/machine-users/:machineUserID", machineUserCtr.UpdateStatus)
	v1RouterGroup.DELETE("/machine-users/:machineUserID", machineUserCtr.Delete)
	// 服务账号角色（服务账号侧授权入口）
	v1RouterGroup.GET("/machine-users/:machineUserID/roles", machineUserCtr.ListRoles)
	v1RouterGroup.PUT("/machine-users/:machineUserID/roles", machineUserCtr.UpdateRoles)
}
