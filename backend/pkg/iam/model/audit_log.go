package model

import (
	"gorm.io/gorm"
)

const TableNameAuditLog = "audit_log"

type AuditLogEntity struct {
	gorm.Model
	ActorPersonID uint   `gorm:"column:actor_person_id;type:bigint unsigned;not null;default 0;comment:操作人person id"`
	ActorUserID   uint   `gorm:"column:actor_user_id;type:bigint unsigned;not null;default 0;comment:操作人user id"`
	TenantID      uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	ClientID      string `gorm:"column:client_id;type:varchar(64);not null;default '';comment:客户端id"`
	Action        string `gorm:"column:action;type:varchar(64);not null;default '';comment:动作标识"`
	TargetType    string `gorm:"column:target_type;type:varchar(64);not null;default '';comment:目标类型"`
	TargetID      uint   `gorm:"column:target_id;type:bigint unsigned;not null;default 0;comment:目标id"`
	Result        string `gorm:"column:result;type:varchar(16);not null;default '';comment:结果 success/failure"`
	IP            string `gorm:"column:ip;type:varchar(64);not null;default '';comment:IP"`
	UserAgent     string `gorm:"column:user_agent;type:varchar(512);not null;default '';comment:UA"`
	Detail        string `gorm:"column:detail;type:text;comment:详情"`
	CreatedBy     uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy     uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy     uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (AuditLogEntity) TableName() string { return TableNameAuditLog }

type AuditLogEntityList []AuditLogEntity

func (l AuditLogEntityList) ToMap() map[uint]AuditLogEntity {
	m := make(map[uint]AuditLogEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
