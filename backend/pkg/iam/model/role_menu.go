package model

import (
	"gorm.io/gorm"
)

const TableNameRoleMenu = "role_menu"

type RoleMenuEntity struct {
	gorm.Model
	TenantID uint `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	RoleID   uint `gorm:"column:role_id;type:bigint unsigned;not null;default 0;comment:角色ID" json:"roleID"`
	MenuID   uint `gorm:"column:menu_id;type:bigint unsigned;not null;default 0;comment:菜单ID" json:"menuID"`
	CreatedBy uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (RoleMenuEntity) TableName() string {
	return TableNameRoleMenu
}

type RoleMenuEntityList []RoleMenuEntity

func (l RoleMenuEntityList) ToMap() map[uint]RoleMenuEntity {
	m := make(map[uint]RoleMenuEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}