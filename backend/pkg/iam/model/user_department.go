package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameUserDepartment = "user_department"

type UserDepartmentEntity struct {
	gormdao.BaseEntity
	TenantID     string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	UserID       string `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID"`
	DepartmentID string `gorm:"column:department_id;type:varchar(36);not null;default:'';comment:部门ID"`
	IsPrimary    int8   `gorm:"column:is_primary;type:smallint;not null;default:0;comment:是否主部门"`
	CreatedBy    string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人ID"`
	UpdatedBy    string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人ID"`
	DeletedBy    string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人ID"`
}

func (UserDepartmentEntity) TableName() string {
	return TableNameUserDepartment
}

type UserDepartmentEntityList []UserDepartmentEntity

func (l UserDepartmentEntityList) ToMap() map[string]UserDepartmentEntity {
	m := make(map[string]UserDepartmentEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
