package router

import (
	"github.com/morehao/ark-iam/iam/platformadmin/controller/ctraudit"
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// auditLogRouter 审计日志只读查询：能进平台控制台即可查（平台侧不区分读/写准入）。
func auditLogRouter(rg *gin.RouterGroup, db gormdao.DBGetter) {
	auditLogCtr := ctraudit.NewAuditLogCtr(db)
	rg.GET("/audit-logs", auditLogCtr.PageList)
}
