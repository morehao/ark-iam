package router

import (
	"github.com/morehao/ark-iam/tenantadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func departmentRouter(groups *ginserver.RouterGroups) {
	departmentCtr := ctrtenant.NewDepartmentCtr()
	departmentUserCtr := ctrtenant.NewDepartmentUserCtr()

	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	// 部门树
	v1RouterGroup.POST("/departments", departmentCtr.Create)
	v1RouterGroup.GET("/departments/tree", departmentCtr.Tree)
	v1RouterGroup.GET("/departments/:departmentID/children", departmentCtr.Children)
	v1RouterGroup.PUT("/departments/:departmentID", departmentCtr.Update)
	v1RouterGroup.PATCH("/departments/:departmentID", departmentCtr.UpdateStatus)
	v1RouterGroup.DELETE("/departments/:departmentID", departmentCtr.Delete)
	// 部门关系
	v1RouterGroup.GET("/departments/:departmentID/users", departmentUserCtr.PageList)
	v1RouterGroup.POST("/departments/:departmentID/users", departmentUserCtr.Create)
	v1RouterGroup.PUT("/departments/:departmentID/users/:userID", departmentUserCtr.Update)
	v1RouterGroup.DELETE("/departments/:departmentID/users/:userID", departmentUserCtr.Delete)
}
