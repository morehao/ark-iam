package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

const TableNameUserIdentity = "user_identity"

type UserIdentityEntity struct {
	gorm.Model
	TenantID         uint            `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	UserID           uint            `gorm:"column:user_id;type:bigint unsigned;not null;default 0;comment:用户ID"`
	ConnectorID      uint            `gorm:"column:connector_id;type:bigint unsigned;not null;default 0;comment:连接器ID"`
	Issuer           string          `gorm:"column:issuer;type:varchar(256);not null;default '';comment:身份提供商"`
	ExternalSubject  string          `gorm:"column:external_subject;type:varchar(128);not null;default '';comment:外部主体标识"`
	Detail           json.RawMessage `gorm:"column:detail;type:json;not null;default '{}';comment:详细信息"`
	CreatedBy        uint            `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人ID"`
	UpdatedBy        uint            `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人ID"`
	DeletedBy        uint            `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人ID"`
}

func (UserIdentityEntity) TableName() string {
	return TableNameUserIdentity
}

type UserIdentityEntityList []UserIdentityEntity

func (l UserIdentityEntityList) ToMap() map[uint]UserIdentityEntity {
	m := make(map[uint]UserIdentityEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
