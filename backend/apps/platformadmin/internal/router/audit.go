package router

import (
	"github.com/morehao/ark-iam/platformadmin/internal/controller/ctraudit"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// auditLogRouter 审计日志只读查询：能进平台控制台即可查（平台侧不区分读/写准入）。
func auditLogRouter(groups *ginserver.RouterGroups) {
	auditLogCtr := ctraudit.NewAuditLogCtr()
	v1RouterGroup := groups.MustGetGroup(ginserver.ApiVersionV1)
	v1RouterGroup.GET("/audit-logs", auditLogCtr.PageList)
}
