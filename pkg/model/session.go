package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameSession = "iam_session"

// SessionAuditEntity 登录会话审计记录：只追加、不可变。
//
// 定位与 user_login_log 一致——会话的事实源是 Redis（撤销走删 key），
// 撤销时间由 refresh_token.revoked_at 承担，因此本表不承载任何状态流转。
// 原 status / revoked_at / last_active_at 三列已下线（见
// docs/design/glossary.md「会话审计」）：status 恒为 active 且无读取路径，
// 另两列从未被写入，留着会让读者误以为可以查 `WHERE status='revoked'`。
type SessionAuditEntity struct {
	gormdao.BaseEntity
	PersonID  string `gorm:"column:person_id;type:varchar(36);not null;default:'';comment:自然人id"`
	SessionID string `gorm:"column:session_id;type:varchar(64);not null;default:'';comment:会话id"`
	// Issuer 会话审计所属 issuer；原 session_id 单列唯一已删除，改由 partial unique 索引
	// iam_idx_session_issuer_sid_unique 约束 (issuer, session_id) WHERE deleted_at IS NULL。
	Issuer    string    `gorm:"column:issuer;type:varchar(256);not null;default:'';comment:所属issuer"`
	TenantID  string    `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	ClientIP  string    `gorm:"column:client_ip;type:varchar(64);not null;default:'';comment:IP"`
	UserAgent string    `gorm:"column:user_agent;type:varchar(512);not null;default:'';comment:UA"`
	LoginTime time.Time `gorm:"column:login_time;not null;default:CURRENT_TIMESTAMP;comment:登录时间"`
	CreatedBy string    `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy string    `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy string    `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (SessionAuditEntity) TableName() string { return TableNameSession }

type SessionAuditEntityList []SessionAuditEntity

func (l SessionAuditEntityList) ToMap() map[string]SessionAuditEntity {
	m := make(map[string]SessionAuditEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
