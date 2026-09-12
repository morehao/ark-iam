package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameRole = "role"

// RoleSource 角色来源。
type RoleSource string

// 角色来源取值（禁止硬编码）。
const (
	RoleSourceBuiltin RoleSource = "builtin" // 内置角色：系统种子数据，禁止删除、禁止改 admin_type
	RoleSourceCustom  RoleSource = "custom"  // 自定义角色：租户管理员创建
)

// SysAdminType 系统管理类型（角色维度的显式能力标签，两值互斥、无高低之分）。
type SysAdminType string

// 系统管理类型取值（禁止硬编码）。
const (
	SysAdminTypeAdmin  SysAdminType = "admin"  // 管理员角色：具备系统管理能力（可管理租户成员/部门/角色/密钥等）
	SysAdminTypeNormal SysAdminType = "normal" // 普通角色：不具备系统管理能力
)

// HasSystemAdmin 判断类型是否具备系统管理能力（仅管理员类型）。
func (t SysAdminType) HasSystemAdmin() bool {
	return t == SysAdminTypeAdmin
}

// IsBuiltinAdmin 判断角色是否为内置管理员（source=builtin 且 admin_type=admin）。
func (r *RoleEntity) IsBuiltinAdmin() bool {
	return r != nil && r.Source == string(RoleSourceBuiltin) && r.AdminType == SysAdminTypeAdmin
}

type RoleEntity struct {
	gormdao.BaseEntity
	TenantID    string       `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID       string       `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id" json:"appID"`
	Name        string       `gorm:"column:name;type:varchar(128);not null;default:'';comment:角色名称" json:"name"`
	Description string       `gorm:"column:description;type:varchar(256);not null;default:'';comment:角色描述" json:"description"`
	Source      string       `gorm:"column:source;type:varchar(16);not null;default:'custom';comment:角色来源(builtin/custom)" json:"source"`
	AdminType   SysAdminType `gorm:"column:admin_type;type:varchar(16);not null;default:'normal';comment:系统管理类型(admin/normal)" json:"adminType"`
	CreatedBy   string       `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy   string       `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy   string       `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
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
