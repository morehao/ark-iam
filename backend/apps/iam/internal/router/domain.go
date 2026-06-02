package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrdomain"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func domainRouter(groups *ginserver.RouterGroups) {
	domainCtr := ctrdomain.NewDomainCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/domain/create", domainCtr.Create)
	v1RouterGroup.POST("/domain/update", domainCtr.Update)
	v1RouterGroup.GET("/domain/detail", domainCtr.Detail)
	v1RouterGroup.POST("/domain/pageList", domainCtr.PageList)
	v1RouterGroup.POST("/domain/delete", domainCtr.Delete)
}