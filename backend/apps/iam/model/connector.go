package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

const TableNameConnector = "connector"

type ConnectorEntity struct {
	gorm.Model
	TenantID           uint           `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	SyncProfile        int8           `gorm:"column:sync_profile;type:tinyint(1);not null;default 0;comment:是否同步资料"`
	EnableTokenStorage int8           `gorm:"column:enable_token_storage;type:tinyint(1);not null;default 0;comment:是否启用令牌存储"`
	ConnectorID        string         `gorm:"column:connector_id;type:varchar(128);not null;default '';comment:连接器ID"`
	Config             json.RawMessage `gorm:"column:config;type:json;not null;default '{}';comment:连接器配置"`
	Metadata           json.RawMessage `gorm:"column:metadata;type:json;not null;default '{}';comment:元数据"`
	CreatedBy          uint           `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人ID"`
	UpdatedBy          uint           `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人ID"`
	DeletedBy          uint           `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人ID"`
}

func (ConnectorEntity) TableName() string {
	return TableNameConnector
}

type ConnectorEntityList []ConnectorEntity

func (l ConnectorEntityList) ToMap() map[uint]ConnectorEntity {
	m := make(map[uint]ConnectorEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}