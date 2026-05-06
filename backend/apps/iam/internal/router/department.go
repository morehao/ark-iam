package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrdepartment"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func departmentRouter(groups *ginserver.RouterGroups) {
	departmentCtr := ctrdepartment.NewDepartmentCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/tenant/create", departmentCtr.Create)
	v1RouterGroup.POST("/tenant/delete", departmentCtr.Delete)
	v1RouterGroup.POST("/tenant/update", departmentCtr.Update)
	v1RouterGroup.GET("/tenant/detail", departmentCtr.Detail)
	v1RouterGroup.POST("/tenant/pageList", departmentCtr.PageList)
	v1RouterGroup.GET("/tenant/tree", departmentCtr.Tree)
}