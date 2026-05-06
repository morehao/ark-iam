package model

import (
	"gorm.io/gorm"
)

const TableNameUserDepartmentRelation = "user_department_relation"

type UserDepartmentRelationEntity struct {
	gorm.Model
	TenantID     uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	UserID       uint   `gorm:"column:user_id;type:bigint unsigned;not null;default 0;comment:用户ID"`
	DepartmentID uint   `gorm:"column:department_id;type:bigint unsigned;not null;default 0;comment:部门ID"`
	IsPrimary    int8   `gorm:"column:is_primary;type:tinyint(1);not null;default 0;comment:是否主部门"`
	CreatedBy    uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人ID"`
	UpdatedBy    uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人ID"`
	DeletedBy    uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人ID"`
}

func (UserDepartmentRelationEntity) TableName() string {
	return TableNameUserDepartmentRelation
}

type UserDepartmentRelationEntityList []UserDepartmentRelationEntity

func (l UserDepartmentRelationEntityList) ToMap() map[uint]UserDepartmentRelationEntity {
	m := make(map[uint]UserDepartmentRelationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}