package router

import (
	"github.com/morehao/ark-iam/iam/tenantadmin/controller/ctrtenant"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func departmentRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	departmentCtr := ctrtenant.NewDepartmentCtr(db)
	departmentUserCtr := ctrtenant.NewDepartmentUserCtr(db)

	// 部门树
	rg.POST("/departments", departmentCtr.Create)
	rg.GET("/departments/tree", departmentCtr.Tree)
	rg.GET("/departments/:departmentID/children", departmentCtr.Children)
	rg.PUT("/departments/:departmentID", departmentCtr.Update)
	rg.PATCH("/departments/:departmentID", departmentCtr.UpdateStatus)
	rg.DELETE("/departments/:departmentID", departmentCtr.Delete)
	// 部门关系
	rg.GET("/departments/:departmentID/users", departmentUserCtr.PageList)
	rg.POST("/departments/:departmentID/users", departmentUserCtr.Create)
	rg.PUT("/departments/:departmentID/users/:userID", departmentUserCtr.Update)
	rg.DELETE("/departments/:departmentID/users/:userID", departmentUserCtr.Delete)
}
