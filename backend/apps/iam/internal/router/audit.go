package router

import (
	"github.com/morehao/ark-iam/iam/internal/controller/ctrlog"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func auditRouter(groups *ginserver.RouterGroups) {
	logCtr := ctrlog.NewLogCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)

	v1RouterGroup.GET("/audit/log/detail", logCtr.Detail)
	v1RouterGroup.POST("/audit/log/pageList", logCtr.PageList)
}