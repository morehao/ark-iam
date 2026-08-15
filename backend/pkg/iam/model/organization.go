package model

import (
	"encoding/json"
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameOrganization = "organization"

type OrganizationEntity struct {
	gormdao.BaseEntity
	TenantID      string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	Name          string          `gorm:"column:name;type:varchar(128);not null;default:'';comment:组织名称" json:"name"`
	Description   string          `gorm:"column:description;type:varchar(256);not null;default:'';comment:组织描述" json:"description"`
	CustomData    json.RawMessage `gorm:"column:custom_data;type:json;not null;default:('{}');comment:自定义数据" json:"customData"`
	IsMFARequired int8            `gorm:"column:is_mfa_required;type:smallint;not null;default:0;comment:是否需要MFA" json:"isMFARequired"`
	CreatedBy     string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy     string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy     string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (OrganizationEntity) TableName() string {
	return TableNameOrganization
}

type OrganizationEntityList []OrganizationEntity

func (l OrganizationEntityList) ToMap() map[string]OrganizationEntity {
	m := make(map[string]OrganizationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
