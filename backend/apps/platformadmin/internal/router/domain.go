package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrdomain"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func domainRouter(groups *ginserver.RouterGroups) {
	domainCtr := ctrdomain.NewDomainCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/platformadmin/domain/create", domainCtr.Create)
	v1RouterGroup.POST("/platformadmin/domain/update", domainCtr.Update)
	v1RouterGroup.GET("/platformadmin/domain/detail", domainCtr.Detail)
	v1RouterGroup.POST("/platformadmin/domain/pageList", domainCtr.PageList)
	v1RouterGroup.POST("/platformadmin/domain/delete", domainCtr.Delete)
}
