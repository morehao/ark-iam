package model

import (
	"gorm.io/gorm"
)

const TableNameRoleScope = "role_scope"

type RoleScopeEntity struct {
	gorm.Model
	TenantID uint `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	RoleID   uint `gorm:"column:role_id;type:bigint unsigned;not null;default 0;comment:角色ID" json:"roleID"`
	ScopeID  uint `gorm:"column:scope_id;type:bigint unsigned;not null;default 0;comment:权限ID" json:"scopeID"`
	CreatedBy uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (RoleScopeEntity) TableName() string {
	return TableNameRoleScope
}

type RoleScopeEntityList []RoleScopeEntity

func (l RoleScopeEntityList) ToMap() map[uint]RoleScopeEntity {
	m := make(map[uint]RoleScopeEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}