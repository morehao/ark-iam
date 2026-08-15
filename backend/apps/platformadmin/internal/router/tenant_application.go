package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrtenantapplication"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantApplicationRouter(groups *ginserver.RouterGroups) {
	ctr := ctrtenantapplication.NewTenantApplicationCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/tenant-applications", ctr.Create)
	v1RouterGroup.GET("/tenant-applications", ctr.PageList)
	v1RouterGroup.GET("/tenant-applications/:tenantAppID", ctr.Detail)
	v1RouterGroup.PUT("/tenant-applications/:tenantAppID", ctr.Update)
	v1RouterGroup.DELETE("/tenant-applications/:tenantAppID", ctr.Delete)
}
