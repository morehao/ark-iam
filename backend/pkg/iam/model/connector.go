package model

import (
	"encoding/json"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameConnector = "connector"

type ConnectorEntity struct {
	gormdao.BaseEntity
	TenantID            string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	Name                string          `gorm:"column:name;type:varchar(128);not null;default:'';comment:连接器名称"`
	DisplayName         string          `gorm:"column:display_name;type:varchar(128);not null;default:'';comment:显示名称"`
	Protocol            string          `gorm:"column:protocol;type:varchar(64);not null;default:'';comment:协议类型"`
	Provider            string          `gorm:"column:provider;type:varchar(128);not null;default:'';comment:提供商"`
	Status              string          `gorm:"column:status;type:varchar(32);not null;default:'';comment:状态"`
	AllowAutoCreateUser int8            `gorm:"column:allow_auto_create_user;type:smallint;not null;default:0;comment:是否允许自动创建用户"`
	AllowAccountLink    int8            `gorm:"column:allow_account_link;type:smallint;not null;default:0;comment:是否允许账号关联"`
	SyncProfile         int8            `gorm:"column:sync_profile;type:smallint;not null;default:0;comment:是否同步资料"`
	EnableTokenStorage  int8            `gorm:"column:enable_token_storage;type:smallint;not null;default:0;comment:是否启用令牌存储"`
	Config              json.RawMessage `gorm:"column:config;type:json;not null;default:'{}';comment:连接器配置"`
	ClaimMapping        json.RawMessage `gorm:"column:claim_mapping;type:json;not null;default:'{}';comment:声明映射"`
	DomainPolicy        json.RawMessage `gorm:"column:domain_policy;type:json;not null;default:'{}';comment:域策略"`
	CreatedBy           string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人ID"`
	UpdatedBy           string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人ID"`
	DeletedBy           string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人ID"`
}

func (ConnectorEntity) TableName() string {
	return TableNameConnector
}

type ConnectorEntityList []ConnectorEntity

func (l ConnectorEntityList) ToMap() map[string]ConnectorEntity {
	m := make(map[string]ConnectorEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
