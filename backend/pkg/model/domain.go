package model

import (
	"database/sql"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameDomain = "domain"

type DomainEntity struct {
	gormdao.BaseEntity
	TenantID   string       `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	Domain     string       `gorm:"column:domain;type:varchar(256);not null;default:'';comment:域名"`
	IsVerified bool         `gorm:"column:is_verified;type:boolean;not null;default:false;comment:是否验证"`
	VerifiedAt sql.NullTime `gorm:"column:verified_at;default:null;comment:验证时间"`
	CreatedBy  string       `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy  string       `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy  string       `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (DomainEntity) TableName() string {
	return TableNameDomain
}

type DomainEntityList []DomainEntity
