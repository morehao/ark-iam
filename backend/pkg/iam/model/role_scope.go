package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameRoleScope = "role_scope"

type RoleScopeEntity struct {
	gormdao.BaseEntity
	TenantID  string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	RoleID    string `gorm:"column:role_id;type:varchar(36);not null;default:'';comment:角色ID" json:"roleID"`
	ScopeID   string `gorm:"column:scope_id;type:varchar(36);not null;default:'';comment:权限ID" json:"scopeID"`
	CreatedBy string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (RoleScopeEntity) TableName() string {
	return TableNameRoleScope
}

type RoleScopeEntityList []RoleScopeEntity

func (l RoleScopeEntityList) ToMap() map[string]RoleScopeEntity {
	m := make(map[string]RoleScopeEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
