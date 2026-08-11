package model

import (
	"gorm.io/gorm"
)

const TableNameOrganizationRole = "organization_role"

type OrganizationRoleEntity struct {
	gorm.Model
	TenantID       uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	OrganizationID uint   `gorm:"column:organization_id;type:bigint unsigned;not null;default 0;comment:组织ID" json:"organizationID"`
	Name           string `gorm:"column:name;type:varchar(128);not null;default '';comment:角色名称" json:"name"`
	Description    string `gorm:"column:description;type:varchar(256);not null;default '';comment:角色描述" json:"description"`
	Type           string `gorm:"column:type;type:varchar(32);not null;default 'User';comment:角色类型" json:"type"`
	CreatedBy      uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy      uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy      uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (OrganizationRoleEntity) TableName() string {
	return TableNameOrganizationRole
}

type OrganizationRoleEntityList []OrganizationRoleEntity

func (l OrganizationRoleEntityList) ToMap() map[uint]OrganizationRoleEntity {
	m := make(map[uint]OrganizationRoleEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}