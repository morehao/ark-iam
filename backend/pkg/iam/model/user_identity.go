package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const TableNameUserIdentity = "user_identity"

type UserIdentityEntity struct {
	gorm.Model
	PersonID        uint            `gorm:"column:person_id;type:bigint unsigned;not null;default 0;comment:自然人ID"`
	ConnectorID     uint            `gorm:"column:connector_id;type:bigint unsigned;not null;default 0;comment:连接器ID"`
	Provider        string          `gorm:"column:provider;type:varchar(128);not null;default '';comment:身份提供商"`
	Issuer          string          `gorm:"column:issuer;type:varchar(256);not null;default '';comment:身份签发方"`
	ExternalSubject string          `gorm:"column:external_subject;type:varchar(128);not null;default '';comment:外部主体标识"`
	Detail          json.RawMessage `gorm:"column:detail;type:json;not null;default '{}';comment:详细信息"`
	LastUsedAt      *time.Time      `gorm:"column:last_used_at;type:datetime;comment:最后使用时间"`
	CreatedBy       uint            `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人ID"`
	UpdatedBy       uint            `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人ID"`
	DeletedBy       uint            `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人ID"`
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
