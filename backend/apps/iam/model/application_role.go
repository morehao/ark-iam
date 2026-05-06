package model

import (
	"gorm.io/gorm"
)

const TableNameApplicationRole = "application_role"

type ApplicationRoleEntity struct {
	gorm.Model
	TenantID      uint `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	ApplicationID uint `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:应用ID" json:"applicationID"`
	RoleID        uint `gorm:"column:role_id;type:bigint unsigned;not null;default 0;comment:角色ID" json:"roleID"`
	CreatedBy     uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy     uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy     uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ApplicationRoleEntity) TableName() string {
	return TableNameApplicationRole
}

type ApplicationRoleEntityList []ApplicationRoleEntity

func (l ApplicationRoleEntityList) ToMap() map[uint]ApplicationRoleEntity {
	m := make(map[uint]ApplicationRoleEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}