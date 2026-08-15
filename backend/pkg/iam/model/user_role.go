package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameUserRole = "user_role"

type UserRoleEntity struct {
	gormdao.BaseEntity
	TenantID  string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	UserID    string `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID" json:"userID"`
	RoleID    string `gorm:"column:role_id;type:varchar(36);not null;default:'';comment:角色ID" json:"roleID"`
	CreatedBy string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (UserRoleEntity) TableName() string {
	return TableNameUserRole
}

type UserRoleEntityList []UserRoleEntity

func (l UserRoleEntityList) ToMap() map[string]UserRoleEntity {
	m := make(map[string]UserRoleEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
