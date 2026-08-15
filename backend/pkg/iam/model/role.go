package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameRole = "role"

type RoleEntity struct {
	gormdao.BaseEntity
	TenantID    string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID       string `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id" json:"appID"`
	Name        string `gorm:"column:name;type:varchar(128);not null;default:'';comment:角色名称" json:"name"`
	Code        string `gorm:"column:code;type:varchar(64);not null;default:'';comment:角色编码" json:"code"`
	Description string `gorm:"column:description;type:varchar(256);not null;default:'';comment:角色描述" json:"description"`
	Type        string `gorm:"column:type;type:varchar(32);not null;default:'User';comment:角色类型" json:"type"`
	IsDefault   bool   `gorm:"column:is_default;type:boolean;not null;default:false;comment:是否默认角色" json:"isDefault"`
	CreatedBy   string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy   string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy   string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (RoleEntity) TableName() string {
	return TableNameRole
}

type RoleEntityList []RoleEntity

func (l RoleEntityList) ToMap() map[string]RoleEntity {
	m := make(map[string]RoleEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
