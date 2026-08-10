package model

import (
	"gorm.io/gorm"
)

const TableNameScope = "scope"

type ScopeEntity struct {
	gorm.Model
	TenantID   uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
	ResourceID uint   `gorm:"column:resource_id;type:bigint unsigned;not null;default 0;comment:资源ID" json:"resourceID"`
	Name       string `gorm:"column:name;type:varchar(256);not null;default '';comment:权限名称" json:"name"`
	Description string `gorm:"column:description;type:text;comment:权限描述" json:"description"`
	CreatedBy  uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy  uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy  uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ScopeEntity) TableName() string {
	return TableNameScope
}

type ScopeEntityList []ScopeEntity

func (l ScopeEntityList) ToMap() map[uint]ScopeEntity {
	m := make(map[uint]ScopeEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}