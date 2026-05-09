package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

const TableNameConnector = "connector"

type ConnectorEntity struct {
	gorm.Model
	TenantID            uint            `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	Name                string          `gorm:"column:name;type:varchar(128);not null;default '';comment:连接器名称"`
	DisplayName         string          `gorm:"column:display_name;type:varchar(128);not null;default '';comment:显示名称"`
	Protocol            string          `gorm:"column:protocol;type:varchar(64);not null;default '';comment:协议类型"`
	Provider            string          `gorm:"column:provider;type:varchar(128);not null;default '';comment:提供商"`
	Status              string          `gorm:"column:status;type:varchar(32);not null;default '';comment:状态"`
	AllowAutoCreateUser int8            `gorm:"column:allow_auto_create_user;type:tinyint(1);not null;default 0;comment:是否允许自动创建用户"`
	AllowAccountLink    int8            `gorm:"column:allow_account_link;type:tinyint(1);not null;default 0;comment:是否允许账号关联"`
	SyncProfile         int8            `gorm:"column:sync_profile;type:tinyint(1);not null;default 0;comment:是否同步资料"`
	EnableTokenStorage  int8            `gorm:"column:enable_token_storage;type:tinyint(1);not null;default 0;comment:是否启用令牌存储"`
	Config              json.RawMessage `gorm:"column:config;type:json;not null;default '{}';comment:连接器配置"`
	ClaimMapping        json.RawMessage `gorm:"column:claim_mapping;type:json;not null;default '{}';comment:声明映射"`
	DomainPolicy        json.RawMessage `gorm:"column:domain_policy;type:json;not null;default '{}';comment:域策略"`
	CreatedBy           uint            `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人ID"`
	UpdatedBy           uint            `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人ID"`
	DeletedBy           uint            `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人ID"`
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
