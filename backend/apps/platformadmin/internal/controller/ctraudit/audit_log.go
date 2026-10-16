package ctraudit

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoaudit"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcaudit"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type AuditLogCtr interface {
	PageList(ctx *gin.Context)
}

type auditLogCtr struct {
	auditLogSvc svcaudit.AuditLogSvc
}

var _ AuditLogCtr = (*auditLogCtr)(nil)

func NewAuditLogCtr() AuditLogCtr {
	return &auditLogCtr{
		auditLogSvc: svcaudit.NewAuditLogSvc(),
	}
}

// @Tags 审计日志
// @Summary 分页查询审计日志
// @Description 平台视角只读查询（跨租户）；tenantID 为空时不限租户，startTime 为空时默认近 7 天，pageSize 上限 100。
// @accept application/json
// @Produce application/json
// @Param req query dtoaudit.AuditLogPageListReq true "审计日志查询条件"
// @Success 200 {object} gincontext.DtoRender{data=dtoaudit.AuditLogPageListResp}
// @Router /v1/platform/audit-logs [get]
func (ctr *auditLogCtr) PageList(ctx *gin.Context) {
	var req dtoaudit.AuditLogPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.auditLogSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
