package model

import (
	"gorm.io/gorm"
)

const TableNameOrganizationRoleUserRelation = "organization_role_user_relation"

type OrganizationRoleUserRelationEntity struct {
	gorm.Model
	TenantID           uint `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	OrganizationID     uint `gorm:"column:organization_id;type:bigint unsigned;not null;default 0;comment:组织ID" json:"organizationID"`
	OrganizationRoleID uint `gorm:"column:organization_role_id;type:bigint unsigned;not null;default 0;comment:组织角色ID" json:"organizationRoleID"`
	UserID             uint `gorm:"column:user_id;type:bigint unsigned;not null;default 0;comment:用户ID" json:"userID"`
	CreatedBy          uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy          uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy          uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (OrganizationRoleUserRelationEntity) TableName() string {
	return TableNameOrganizationRoleUserRelation
}

type OrganizationRoleUserRelationEntityList []OrganizationRoleUserRelationEntity

func (l OrganizationRoleUserRelationEntityList) ToMap() map[uint]OrganizationRoleUserRelationEntity {
	m := make(map[uint]OrganizationRoleUserRelationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}