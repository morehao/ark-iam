package model

import (
	"encoding/json"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameSystem = "system"

type SystemEntity struct {
	gormdao.BaseEntity
	TenantID  string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	Key       string          `gorm:"column:key;type:varchar(256);not null;default:'';comment:配置键"`
	Value     json.RawMessage `gorm:"column:value;type:json;not null;default:'{}';comment:配置值"`
	CreatedBy string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (SystemEntity) TableName() string {
	return TableNameSystem
}

type SystemEntityList []SystemEntity

func (l SystemEntityList) ToMap() map[string]SystemEntity {
	m := make(map[string]SystemEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
