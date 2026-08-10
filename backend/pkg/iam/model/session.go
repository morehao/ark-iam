package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameSession = "session"

type SessionAuditEntity struct {
	gorm.Model
	PersonID     uint       `gorm:"column:person_id;type:bigint unsigned;not null;default 0;comment:自然人id"`
	SessionID    string     `gorm:"column:session_id;type:varchar(64);not null;default '';uniqueIndex;comment:会话id"`
	TenantID     uint       `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	ClientIP     string     `gorm:"column:client_ip;type:varchar(64);not null;default '';comment:IP"`
	UserAgent    string     `gorm:"column:user_agent;type:varchar(512);not null;default '';comment:UA"`
	LoginTime    time.Time  `gorm:"column:login_time;type:datetime;not null;default CURRENT_TIMESTAMP;comment:登录时间"`
	LastActiveAt *time.Time `gorm:"column:last_active_at;type:datetime;comment:最后活跃"`
	RevokedAt    *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间"`
	Status       string     `gorm:"column:status;type:varchar(16);not null;default 'active';comment:active/revoked"`
	CreatedBy    uint       `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy    uint       `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy    uint       `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (SessionAuditEntity) TableName() string { return TableNameSession }

type SessionAuditEntityList []SessionAuditEntity

func (l SessionAuditEntityList) ToMap() map[uint]SessionAuditEntity {
	m := make(map[uint]SessionAuditEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
