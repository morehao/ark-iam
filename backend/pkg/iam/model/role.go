package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameRole = "role"

// RoleSource 角色来源。
type RoleSource string

// 角色来源取值（禁止硬编码）。
const (
	RoleSourceBuiltin RoleSource = "builtin" // 内置角色：系统种子数据，禁止删除/改核心字段
	RoleSourceCustom  RoleSource = "custom"  // 自定义角色：租户管理员创建
)

// SysAdminLevel 系统管理等级（单调门槛：等级越高，可管的系统功能越多）。
type SysAdminLevel string

// 系统管理等级取值（禁止硬编码）。
const (
	SysAdminLevelMember SysAdminLevel = "member" // 普通租户成员（无系统管理能力）
	SysAdminLevelSuper  SysAdminLevel = "super"  // 超级管理员（全部系统功能）
)

// HasSystemAdmin 判断等级是否具备系统管理能力（仅 super）。
func (l SysAdminLevel) HasSystemAdmin() bool {
	return l == SysAdminLevelSuper
}

// SysAdminRank 返回系统管理等级的序数（用于门槛比较：member < super）。
func (l SysAdminLevel) SysAdminRank() int {
	if l == SysAdminLevelSuper {
		return 2
	}
	return 1 // SysAdminLevelMember
}

// IsBuiltinAdmin 判断角色是否为内置管理员（source=builtin 且 admin_level=super）。
func (r *RoleEntity) IsBuiltinAdmin() bool {
	return r != nil && r.Source == string(RoleSourceBuiltin) && SysAdminLevel(r.AdminLevel) == SysAdminLevelSuper
}

type RoleEntity struct {
	gormdao.BaseEntity
	TenantID    string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID       string `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id" json:"appID"`
	Name        string `gorm:"column:name;type:varchar(128);not null;default:'';comment:角色名称" json:"name"`
	Code        string `gorm:"column:code;type:varchar(64);not null;default:'';comment:角色编码" json:"code"`
	Description string `gorm:"column:description;type:varchar(256);not null;default:'';comment:角色描述" json:"description"`
	Source      string `gorm:"column:source;type:varchar(16);not null;default:'custom';comment:角色来源(builtin/custom)" json:"source"`
	AdminLevel  string `gorm:"column:admin_level;type:varchar(16);not null;default:'member';comment:系统管理等级(member/super)" json:"adminLevel"`
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
