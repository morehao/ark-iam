package model

import (
	"encoding/json"
	"gorm.io/gorm"
)

const TableNameApplication = "application"

type ApplicationEntity struct {
	gorm.Model
	TenantID              uint                `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	Name                  string              `gorm:"column:name;type:varchar(256);not null;default '';comment:应用名称" json:"name"`
	Secret                string              `gorm:"column:secret;type:varchar(64);not null;default '';comment:应用密钥" json:"secret"`
	Description           string              `gorm:"column:description;type:text;comment:应用描述" json:"description"`
	Type                  string              `gorm:"column:type;type:varchar(32);not null;default '';comment:应用类型" json:"type"`
	OIDCClientMetadata    json.RawMessage     `gorm:"column:oidc_client_metadata;type:json;not null;default ('{}');comment:OIDC客户端配置" json:"oidcClientMetadata"`
	CustomClientMetadata  json.RawMessage     `gorm:"column:custom_client_metadata;type:json;not null;default ('{}');comment:自定义客户端配置" json:"customClientMetadata"`
	IsThirdParty          int8                `gorm:"column:is_third_party;type:tinyint(1);not null;default 0;comment:是否第三方应用" json:"isThirdParty"`
	CreatedBy             uint                `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy             uint                `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy             uint                `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ApplicationEntity) TableName() string {
	return TableNameApplication
}

type ApplicationEntityList []ApplicationEntity

func (l ApplicationEntityList) ToMap() map[uint]ApplicationEntity {
	m := make(map[uint]ApplicationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}