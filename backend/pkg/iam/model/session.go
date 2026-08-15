package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameSession = "session"

type SessionAuditEntity struct {
	gormdao.BaseEntity
	PersonID     string     `gorm:"column:person_id;type:varchar(36);not null;default:'';comment:自然人id"`
	SessionID    string     `gorm:"column:session_id;type:varchar(64);not null;default:'';uniqueIndex;comment:会话id"`
	TenantID     string     `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	ClientIP     string     `gorm:"column:client_ip;type:varchar(64);not null;default:'';comment:IP"`
	UserAgent    string     `gorm:"column:user_agent;type:varchar(512);not null;default:'';comment:UA"`
	LoginTime    time.Time  `gorm:"column:login_time;not null;default:CURRENT_TIMESTAMP;comment:登录时间"`
	LastActiveAt *time.Time `gorm:"column:last_active_at;comment:最后活跃"`
	RevokedAt    *time.Time `gorm:"column:revoked_at;comment:撤销时间"`
	Status       string     `gorm:"column:status;type:varchar(16);not null;default:'active';comment:active/revoked"`
	CreatedBy    string     `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy    string     `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy    string     `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
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
