package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrtenant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantRouter(groups *ginserver.RouterGroups) {
	tenantCtr := ctrtenant.NewTenantCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/tenants", tenantCtr.Create)
	v1RouterGroup.GET("/tenants", tenantCtr.PageList)
	v1RouterGroup.GET("/tenants/:tenantID", tenantCtr.Detail)
	v1RouterGroup.PUT("/tenants/:tenantID", tenantCtr.Update)
	v1RouterGroup.DELETE("/tenants/:tenantID", tenantCtr.Delete)
}

func logRouter(groups *ginserver.RouterGroups) {
	logCtr := ctrtenant.NewLogCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/logs", logCtr.PageList)
	v1RouterGroup.GET("/logs/:logID", logCtr.Detail)
}
