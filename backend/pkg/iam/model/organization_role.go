package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameOrganizationRole = "organization_role"

type OrganizationRoleEntity struct {
	gormdao.BaseEntity
	TenantID       string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	OrganizationID string `gorm:"column:organization_id;type:varchar(36);not null;default:'';comment:组织ID" json:"organizationID"`
	Name           string `gorm:"column:name;type:varchar(128);not null;default:'';comment:角色名称" json:"name"`
	Description    string `gorm:"column:description;type:varchar(256);not null;default:'';comment:角色描述" json:"description"`
	Type           string `gorm:"column:type;type:varchar(32);not null;default:'User';comment:角色类型" json:"type"`
	CreatedBy      string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy      string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy      string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (OrganizationRoleEntity) TableName() string {
	return TableNameOrganizationRole
}

type OrganizationRoleEntityList []OrganizationRoleEntity

func (l OrganizationRoleEntityList) ToMap() map[string]OrganizationRoleEntity {
	m := make(map[string]OrganizationRoleEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
