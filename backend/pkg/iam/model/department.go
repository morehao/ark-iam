package model

import (
	"gorm.io/gorm"
)

const TableNameDepartment = "department"

type DepartmentEntity struct {
	gorm.Model
	TenantID     uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	ParentID     uint   `gorm:"column:parent_id;type:bigint unsigned;not null;default 0;comment:父部门ID"`
	Name         string `gorm:"column:name;type:varchar(128);not null;default '';comment:部门名称"`
	Code         string `gorm:"column:code;type:varchar(64);not null;default '';comment:部门编码"`
	Sort         int    `gorm:"column:sort;type:int;not null;default 0;comment:排序"`
	LeaderUserID uint   `gorm:"column:leader_user_id;type:bigint unsigned;not null;default 0;comment:部门负责人用户ID"`
	CreatedBy    uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy    uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy    uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (DepartmentEntity) TableName() string {
	return TableNameDepartment
}

type DepartmentEntityList []DepartmentEntity

func (l DepartmentEntityList) ToMap() map[uint]DepartmentEntity {
	m := make(map[uint]DepartmentEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}