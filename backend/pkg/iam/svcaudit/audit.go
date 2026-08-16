package svcaudit

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

const (
	ActionLogin                         = "login"
	ActionLogout                        = "logout"
	ActionTenantSwitch                  = "tenant.switch"
	ActionTenantCreate                  = "tenant.create"
	ActionApplicationCreate             = "application.create"
	ActionApplicationClientCreate       = "application_client.create"
	ActionApplicationClientCreateSecret = "application_client.create_secret"
	ActionApiKeyCreate                  = "api_key.create"
	ActionApiKeyRevoke                  = "api_key.revoke"
)

type AuditEntry struct {
	Action     string
	TenantID   string
	TargetType string
	TargetID   string
	Result     string // success / failure
	Detail     string
	ClientID   string
}

var newAuditLogDao = func() *dao.AuditLogDao { return dao.NewAuditLogDao() }

func WriteAudit(ctx *gin.Context, e AuditEntry) {
	if ctx == nil || ctx.Request == nil {
		return
	}
	// 审计写入是 best-effort：任何异常（含 DB 未初始化导致 nil *gorm.DB panic）
	// 都不得阻断业务主流程，统一 recover 后仅记日志。
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf(ctx, "[svcaudit.WriteAudit] panic recovered, action:%s, panic:%v", e.Action, r)
		}
	}()
	entity := &model.AuditLogEntity{
		ActorPersonID: ctx.GetString(gcontext.KeyPersonID),
		ActorUserID:   ctx.GetString(gcontext.KeyUserID),
		TenantID:      e.TenantID,
		ClientID:      e.ClientID,
		Action:        e.Action,
		TargetType:    e.TargetType,
		TargetID:      e.TargetID,
		Result:        e.Result,
		IP:            gincontext.GetClientIP(ctx),
		UserAgent:     ctx.GetHeader("User-Agent"),
		Detail:        e.Detail,
		CreatedBy:     ctx.GetString(gcontext.KeyUserID),
	}
	if err := newAuditLogDao().Insert(context.Background(), entity); err != nil {
		glog.Errorf(ctx, "[svcaudit.WriteAudit] failed, action:%s, err:%v", e.Action, err)
	}
}
