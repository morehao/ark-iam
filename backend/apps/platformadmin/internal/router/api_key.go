package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctrapikey"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func apiKeyRouter(groups *ginserver.RouterGroups) {
	apiKeyCtr := ctrapikey.NewApiKeyCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	// 平台侧仅保留跨租户只读监督；创建/吊销/删除等写动作归属租户端 /v1/tenant/api-keys
	v1RouterGroup.GET("/api-keys/supervision", apiKeyCtr.PageListSupervision)
}
