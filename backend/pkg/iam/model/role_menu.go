package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameRoleMenu = "role_menu"

type RoleMenuEntity struct {
	gormdao.BaseEntity
	TenantID  string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	RoleID    string `gorm:"column:role_id;type:varchar(36);not null;default:'';comment:角色ID" json:"roleID"`
	MenuID    string `gorm:"column:menu_id;type:varchar(36);not null;default:'';comment:菜单ID" json:"menuID"`
	CreatedBy string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (RoleMenuEntity) TableName() string {
	return TableNameRoleMenu
}

type RoleMenuEntityList []RoleMenuEntity

func (l RoleMenuEntityList) ToMap() map[string]RoleMenuEntity {
	m := make(map[string]RoleMenuEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
