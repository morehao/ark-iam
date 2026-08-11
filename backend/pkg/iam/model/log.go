package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

const TableNameLog = "log"

type LogEntity struct {
	gorm.Model
	TenantID uint           `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	Key      string         `gorm:"column:key;type:varchar(128);not null;default '';comment:日志键"`
	Payload  json.RawMessage `gorm:"column:payload;type:json;not null;default '{}';comment:日志内容"`
}

func (LogEntity) TableName() string {
	return TableNameLog
}

type LogEntityList []LogEntity

func (l LogEntityList) ToMap() map[uint]LogEntity {
	m := make(map[uint]LogEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}