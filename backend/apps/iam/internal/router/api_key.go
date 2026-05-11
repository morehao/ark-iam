package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrapikey"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func apiKeyRouter(groups *ginserver.RouterGroups) {
	apiKeyCtr := ctrapikey.NewApiKeyCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.POST("/apiKey/create", apiKeyCtr.Create)
	v1RouterGroup.POST("/apiKey/pageList", apiKeyCtr.PageList)
	v1RouterGroup.POST("/apiKey/revoke", apiKeyCtr.Revoke)
	v1RouterGroup.POST("/apiKey/delete", apiKeyCtr.Delete)
}