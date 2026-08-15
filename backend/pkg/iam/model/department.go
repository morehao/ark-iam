package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameDepartment = "department"

type DepartmentEntity struct {
	gormdao.BaseEntity
	TenantID     string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	ParentID     string `gorm:"column:parent_id;type:varchar(36);not null;default:'';comment:父部门ID"`
	Name         string `gorm:"column:name;type:varchar(128);not null;default:'';comment:部门名称"`
	Code         string `gorm:"column:code;type:varchar(64);not null;default:'';comment:部门编码"`
	Sort         int    `gorm:"column:sort;type:int;not null;default:0;comment:排序"`
	LeaderUserID string `gorm:"column:leader_user_id;type:varchar(36);not null;default:'';comment:部门负责人用户ID"`
	CreatedBy    string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy    string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy    string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (DepartmentEntity) TableName() string {
	return TableNameDepartment
}

type DepartmentEntityList []DepartmentEntity

func (l DepartmentEntityList) ToMap() map[string]DepartmentEntity {
	m := make(map[string]DepartmentEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
