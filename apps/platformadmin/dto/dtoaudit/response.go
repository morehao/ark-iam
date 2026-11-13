package dtoaudit

import "github.com/morehao/ark-iam/pkg/model"

// AuditLogPageListItem 审计日志列表项。
//
// 时间只回传 CreatedAt：审计是追加型记录，updated_at 恒等于 created_at，展示无意义
// （前端规范 R3「纯追加型记录不设更新时间列」），故不提供 createdAt/updatedAt 双列。
type AuditLogPageListItem struct {
	AuditLogID    string                `json:"auditLogID"`    // 审计日志ID
	ActorPersonID string                `json:"actorPersonID"` // 操作人自然人ID
	ActorUserID   string                `json:"actorUserID"`   // 操作人租户成员ID
	TenantID      string                `json:"tenantID"`      // 租户ID
	ClientID      string                `json:"clientID"`      // 客户端ID（OIDC client_id）
	Action        model.AuditAction     `json:"action"`        // 动作标识
	TargetType    model.AuditTargetType `json:"targetType"`    // 目标类型
	TargetID      string                `json:"targetID"`      // 目标ID
	Result        model.AuditResult     `json:"result"`        // 结果 success/failure
	IP            string                `json:"ip"`            // 客户端IP
	UserAgent     string                `json:"userAgent"`     // 用户代理
	Detail        string                `json:"detail"`        // 详情
	CreatedAt     int64                 `json:"createdAt"`     // 发生时间(unix 秒)
}

type AuditLogPageListResp struct {
	List  []AuditLogPageListItem `json:"list"`  // 数据列表
	Total int64                  `json:"total"` // 数据总条数
}
