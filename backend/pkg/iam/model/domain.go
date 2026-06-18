package model

import (
	"database/sql"

	"gorm.io/gorm"
)

const TableNameDomain = "domain"

type DomainEntity struct {
	gorm.Model
	TenantID   uint         `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	Domain     string       `gorm:"column:domain;type:varchar(256);not null;default '';comment:域名"`
	IsVerified int8         `gorm:"column:is_verified;type:tinyint(1);not null;default 0;comment:是否验证"`
	VerifiedAt sql.NullTime `gorm:"column:verified_at;type:datetime;default null;comment:验证时间"`
	CreatedBy  uint         `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy  uint         `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy  uint         `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (DomainEntity) TableName() string {
	return TableNameDomain
}

type DomainEntityList []DomainEntity
