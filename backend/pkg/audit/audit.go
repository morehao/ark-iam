package audit

import (
	"context"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gcontext"
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
	ActionInstallationInitialize        = model.AuditActionInstallationInitialize
)

type AuditEntry struct {
	Action     model.AuditAction
	TenantID   string
	TargetType model.AuditTargetType
	TargetID   string
	Result     model.AuditResult
	Detail     string
	ClientID   string
	// Operator 是发起本次操作的主体与请求元数据，由调用方从 DTO（req.Operator）
	// 或显式参数填入。审计**不再**读 gin 上下文：服务层只收 context.Context。
	//
	// IP/UserAgent 另有显式字段可覆盖 Operator 里的值（用于非 HTTP 入口，
	// 如后台任务、协议层回调这类拿不到 gin 请求的场景）。
	Operator objidentity.Operator
	// IP / UserAgent 显式覆盖 Operator.ClientIP / Operator.UserAgent；
	// 留空则取 Operator 的值。
	IP        string
	UserAgent string
}

var newAuditLogDao = func() *dao.AuditLogDao { return dao.NewAuditLogDao() }

// WriteAudit 写入一条审计日志（best-effort，失败不阻断业务）。
//
// ctx 是纯 context.Context：审计的身份与 IP/UA 全部来自 e.Operator，
// 不再从 gin 上下文读取（身份是业务输入，走 DTO）。
func WriteAudit(ctx context.Context, e AuditEntry) {
	if ctx == nil {
		return
	}
	// 审计写入是 best-effort：任何异常（含 DB 未初始化导致 nil *gorm.DB panic）
	// 都不得阻断业务主流程，统一 recover 后仅记日志。
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf(ctx, "[audit.WriteAudit] panic recovered, action:%s, panic:%v", e.Action, r)
		}
	}()
	ip := e.Operator.ClientIP
	if e.IP != "" {
		ip = e.IP
	}
	userAgent := e.Operator.UserAgent
	if e.UserAgent != "" {
		userAgent = e.UserAgent
	}
	entity := &model.AuditLogEntity{
		ActorPersonID: e.Operator.PersonID,
		ActorUserID:   e.Operator.UserID,
		TenantID:      e.TenantID,
		ClientID:      e.ClientID,
		Action:        e.Action,
		TargetType:    e.TargetType,
		TargetID:      e.TargetID,
		Result:        e.Result,
		IP:            ip,
		UserAgent:     userAgent,
		Detail:        e.Detail,
		// created_by 与 ActorUserID 同口径（成员主键）；未携带 operator 时落空串。
		CreatedBy: e.Operator.UserID,
	}
	// 审计是跨租户旁路写入（实体自带 TenantID）：显式声明「全部租户」作用域。
	// 不用缺失作用域表达跨租户，否则 fail-closed 下审计写入会整体失败。
	if err := newAuditLogDao().Insert(gcontext.WithTenantScope(context.Background(), gcontext.AllScope()), entity); err != nil {
		glog.Errorf(ctx, "[audit.WriteAudit] failed, action:%s, err:%v", e.Action, err)
	}
}
