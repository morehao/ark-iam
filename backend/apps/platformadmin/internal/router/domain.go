package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrdomain"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func domainRouter(groups *ginserver.RouterGroups) {
	domainCtr := ctrdomain.NewDomainCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/domains", domainCtr.Create)
	v1RouterGroup.GET("/domains", domainCtr.PageList)
	v1RouterGroup.GET("/domains/:domainID", domainCtr.Detail)
	v1RouterGroup.PUT("/domains/:domainID", domainCtr.Update)
	v1RouterGroup.DELETE("/domains/:domainID", domainCtr.Delete)
}
