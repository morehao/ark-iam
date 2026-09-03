package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrapikey"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func apiKeyRouter(groups *ginserver.RouterGroups) {
	apiKeyCtr := ctrapikey.NewApiKeyCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.POST("/api-keys", apiKeyCtr.Create)
	v1RouterGroup.GET("/api-keys", apiKeyCtr.PageList)
	v1RouterGroup.GET("/api-keys/supervision", apiKeyCtr.PageListSupervision)
	v1RouterGroup.POST("/api-keys/:apiKeyID/revoke", apiKeyCtr.Revoke)
	v1RouterGroup.DELETE("/api-keys/:apiKeyID", apiKeyCtr.Delete)
}
