package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrdepartment"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrorganization"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrsystem"
	"github.com/morehao/ark-iam/iam/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()
	departmentCtr := ctrdepartment.NewDepartmentCtr()
	organizationCtr := ctrorganization.NewOrganizationCtr()
	systemCtr := ctrsystem.NewSystemCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/tenant/create", tenantCtr.Create)
	v1RouterGroup.POST("/tenant/delete", tenantCtr.Delete)
	v1RouterGroup.POST("/tenant/update", tenantCtr.Update)
	v1RouterGroup.GET("/tenant/detail", tenantCtr.Detail)
	v1RouterGroup.POST("/tenant/pageList", tenantCtr.PageList)

	v1RouterGroup.POST("/department/create", departmentCtr.Create)
	v1RouterGroup.POST("/department/delete", departmentCtr.Delete)
	v1RouterGroup.POST("/department/update", departmentCtr.Update)
	v1RouterGroup.GET("/department/detail", departmentCtr.Detail)
	v1RouterGroup.POST("/department/pageList", departmentCtr.PageList)
	v1RouterGroup.GET("/department/tree", departmentCtr.Tree)

	v1RouterGroup.POST("/organization/create", organizationCtr.Create)
	v1RouterGroup.POST("/organization/delete", organizationCtr.Delete)
	v1RouterGroup.POST("/organization/update", organizationCtr.Update)
	v1RouterGroup.GET("/organization/detail", organizationCtr.Detail)
	v1RouterGroup.POST("/organization/pageList", organizationCtr.PageList)

	v1RouterGroup.POST("/system/create", systemCtr.Create)
	v1RouterGroup.POST("/system/delete", systemCtr.Delete)
	v1RouterGroup.POST("/system/update", systemCtr.Update)
	v1RouterGroup.GET("/system/detail", systemCtr.Detail)
	v1RouterGroup.POST("/system/pageList", systemCtr.PageList)
}