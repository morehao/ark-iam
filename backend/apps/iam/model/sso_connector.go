package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

const TableNameSsoConnector = "sso_connector"

type SsoConnectorEntity struct {
	gorm.Model
	TenantID           uint           `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	ProviderName       string         `gorm:"column:provider_name;type:varchar(128);not null;default '';comment:提供商名称"`
	ConnectorName      string         `gorm:"column:connector_name;type:varchar(128);not null;default '';comment:连接器名称"`
	Config             json.RawMessage `gorm:"column:config;type:json;not null;default '{}';comment:配置"`
	Domains            json.RawMessage `gorm:"column:domains;type:json;not null;default '[]';comment:域名列表"`
	Branding           json.RawMessage `gorm:"column:branding;type:json;not null;default '{}';comment:品牌配置"`
	SyncProfile        int8           `gorm:"column:sync_profile;type:tinyint(1);not null;default 0;comment:是否同步资料"`
	EnableTokenStorage int8           `gorm:"column:enable_token_storage;type:tinyint(1);not null;default 0;comment:是否启用令牌存储"`
	CreatedBy          uint           `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人ID"`
	UpdatedBy          uint           `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人ID"`
	DeletedBy          uint           `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人ID"`
}

func (SsoConnectorEntity) TableName() string {
	return TableNameSsoConnector
}

type SsoConnectorEntityList []SsoConnectorEntity

func (l SsoConnectorEntityList) ToMap() map[uint]SsoConnectorEntity {
	m := make(map[uint]SsoConnectorEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}