package model

import (
	"gorm.io/gorm"
)

const TableNameUserRole = "user_role"

type UserRoleEntity struct {
	gorm.Model
	TenantID uint `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	UserID   uint `gorm:"column:user_id;type:bigint unsigned;not null;default 0;comment:用户ID" json:"userID"`
	RoleID   uint `gorm:"column:role_id;type:bigint unsigned;not null;default 0;comment:角色ID" json:"roleID"`
	CreatedBy uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (UserRoleEntity) TableName() string {
	return TableNameUserRole
}

type UserRoleEntityList []UserRoleEntity

func (l UserRoleEntityList) ToMap() map[uint]UserRoleEntity {
	m := make(map[uint]UserRoleEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}