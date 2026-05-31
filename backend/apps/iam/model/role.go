package model

import (
	"gorm.io/gorm"
)

const TableNameRole = "role"

type RoleEntity struct {
	gorm.Model
	TenantID        uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	ApplicationID   uint   `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:所属应用id" json:"applicationID"`
	Name        string `gorm:"column:name;type:varchar(128);not null;default '';comment:角色名称" json:"name"`
	Code        string `gorm:"column:code;type:varchar(64);not null;default '';comment:角色编码" json:"code"`
	Description string `gorm:"column:description;type:varchar(256);not null;default '';comment:角色描述" json:"description"`
	Type        string `gorm:"column:type;type:varchar(32);not null;default 'User';comment:角色类型" json:"type"`
	IsDefault   int8   `gorm:"column:is_default;type:tinyint(1);not null;default 0;comment:是否默认角色" json:"isDefault"`
	CreatedBy   uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy   uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy   uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (RoleEntity) TableName() string {
	return TableNameRole
}

type RoleEntityList []RoleEntity

func (l RoleEntityList) ToMap() map[uint]RoleEntity {
	m := make(map[uint]RoleEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}