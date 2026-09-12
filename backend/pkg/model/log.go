package model

import (
	"encoding/json"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameLog = "log"

type LogEntity struct {
	gormdao.BaseEntity
	TenantID string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	Key      string          `gorm:"column:key;type:varchar(128);not null;default:'';comment:日志键"`
	Payload  json.RawMessage `gorm:"column:payload;type:json;not null;default:'{}';comment:日志内容"`
}

func (LogEntity) TableName() string {
	return TableNameLog
}

type LogEntityList []LogEntity

func (l LogEntityList) ToMap() map[string]LogEntity {
	m := make(map[string]LogEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
