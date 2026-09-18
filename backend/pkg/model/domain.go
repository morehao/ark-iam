package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameDomain = "domain"

// DomainVerificationStatus 域名验证状态（domain.verification_status）。
// 预留 pending/failed：接入真实验证流程时只加常量，不改列类型、不破坏 API。
type DomainVerificationStatus string

// 域名验证状态取值（禁止硬编码）。
const (
	DomainVerificationUnverified DomainVerificationStatus = "unverified"
	DomainVerificationVerified   DomainVerificationStatus = "verified"
)

type DomainEntity struct {
	gormdao.BaseEntity
	TenantID           string                   `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	Domain             string                   `gorm:"column:domain;type:varchar(256);not null;default:'';comment:域名"`
	VerificationStatus DomainVerificationStatus `gorm:"column:verification_status;type:varchar(16);not null;default:'unverified';comment:验证状态(unverified未验证/verified已验证)"`
	CreatedBy          string                   `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy          string                   `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy          string                   `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (DomainEntity) TableName() string {
	return TableNameDomain
}

type DomainEntityList []DomainEntity
