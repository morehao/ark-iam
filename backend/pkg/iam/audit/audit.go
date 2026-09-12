package audit

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

// 审计动作短名别名：取值事实源在 model.AuditAction*（落库枚举定义在 model 层），
// 此处仅保留 audit.ActionXxx 写法供调用方复用，禁止再出现裸字面量。
const (
	ActionLogin                         = model.AuditActionLogin
	ActionLogout                        = model.AuditActionLogout
	ActionTenantSwitch                  = model.AuditActionTenantSwitch
	ActionTenantCreate                  = model.AuditActionTenantCreate
	ActionTenantAdminPasswordReset      = model.AuditActionTenantAdminPasswordReset
	ActionApplicationCreate             = model.AuditActionApplicationCreate
	ActionApplicationClientCreate       = model.AuditActionApplicationClientCreate
	ActionApplicationClientCreateSecret = model.AuditActionApplicationClientCreateSecret
	ActionApiKeyCreate                  = model.AuditActionApiKeyCreate
	ActionApiKeyRevoke                  = model.AuditActionApiKeyRevoke
)

type AuditEntry struct {
	Action     model.AuditAction
	TenantID   string
	TargetType model.AuditTargetType
	TargetID   string
	Result     model.AuditResult
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
			glog.Errorf(ctx, "[audit.WriteAudit] panic recovered, action:%s, panic:%v", e.Action, r)
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
		glog.Errorf(ctx, "[audit.WriteAudit] failed, action:%s, err:%v", e.Action, err)
	}
}
