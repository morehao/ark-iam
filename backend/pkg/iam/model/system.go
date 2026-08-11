package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

const TableNameSystem = "system"

type SystemEntity struct {
	gorm.Model
	TenantID  uint           `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	Key       string         `gorm:"column:key;type:varchar(256);not null;default '';comment:配置键"`
	Value     json.RawMessage `gorm:"column:value;type:json;not null;default '{}';comment:配置值"`
	CreatedBy uint           `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy uint           `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy uint           `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (SystemEntity) TableName() string {
	return TableNameSystem
}

type SystemEntityList []SystemEntity

func (l SystemEntityList) ToMap() map[uint]SystemEntity {
	m := make(map[uint]SystemEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}