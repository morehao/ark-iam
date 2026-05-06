package model

import (
	"encoding/json"
	"gorm.io/gorm"
)

const TableNameOrganization = "organization"

type OrganizationEntity struct {
	gorm.Model
	TenantID      uint          `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	Name          string        `gorm:"column:name;type:varchar(128);not null;default '';comment:组织名称" json:"name"`
	Description   string        `gorm:"column:description;type:varchar(256);not null;default '';comment:组织描述" json:"description"`
	CustomData    json.RawMessage `gorm:"column:custom_data;type:json;not null;default ('{}');comment:自定义数据" json:"customData"`
	IsMFARequired int8          `gorm:"column:is_mfa_required;type:tinyint(1);not null;default 0;comment:是否需要MFA" json:"isMFARequired"`
	CreatedBy     uint          `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy     uint          `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy     uint          `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (OrganizationEntity) TableName() string {
	return TableNameOrganization
}

type OrganizationEntityList []OrganizationEntity

func (l OrganizationEntityList) ToMap() map[uint]OrganizationEntity {
	m := make(map[uint]OrganizationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}