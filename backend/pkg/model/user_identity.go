package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameUserIdentity = "user_identity"

// EmailVerificationState 第三方身份回传的邮箱验证态（user_identity.detail.emailVerified）。
type EmailVerificationState string

// 邮箱验证态取值（禁止硬编码）。
const (
	EmailVerificationVerified   EmailVerificationState = "verified"
	EmailVerificationUnverified EmailVerificationState = "unverified"
)

type UserIdentityEntity struct {
	gormdao.BaseEntity
	PersonID        string             `gorm:"column:person_id;type:varchar(36);not null;default:'';comment:自然人ID"`
	ConnectorID     string             `gorm:"column:connector_id;type:varchar(36);not null;default:'';comment:连接器ID"`
	Provider        ConnectorProvider  `gorm:"column:provider;type:varchar(128);not null;default:'';comment:身份提供商"`
	Issuer          string             `gorm:"column:issuer;type:varchar(256);not null;default:'';comment:身份签发方"`
	ExternalSubject string             `gorm:"column:external_subject;type:varchar(128);not null;default:'';comment:外部主体标识"`
	Detail          UserIdentityDetail `gorm:"column:detail;type:json;serializer:json;not null;default:'{}';comment:详细信息"`
	CreatedBy       string             `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人ID"`
	UpdatedBy       string             `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人ID"`
	DeletedBy       string             `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人ID"`
}

// 注：同一外部主体（issuer + external_subject）在本系统只能绑定到一个自然人，
// 唯一性由部分唯一索引保证（WHERE deleted_at IS NULL，见 automigrate.go
// EnsurePartialUniqueIndexes），并发回调不会重复创建 person/identity。

func (UserIdentityEntity) TableName() string {
	return TableNameUserIdentity
}

type UserIdentityEntityList []UserIdentityEntity

func (l UserIdentityEntityList) ToMap() map[string]UserIdentityEntity {
	m := make(map[string]UserIdentityEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
