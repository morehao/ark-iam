package model

import (
	"regexp"

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

// RoleCode 角色编码：该角色在**跨系统授权**中的稳定标识，也是 OIDC ID token `groups`
// 声明的取值（下游系统按「前缀 + 角色编码」认策略名，如 RustFS 的 claim_prefix）。
//
// 与 role.name 刻意分离：name 是中文可读展示、归运维改名；code 是契约值，改名即改变下游授权，
// 故仅允许合法形状与租户内唯一，控制台改动需显式提示（谁按它认行 = 下游系统）。
type RoleCode string

// 内置角色编码取值（禁止硬编码）。取值刻意中性：不写下游产品语义（如 consoleAdmin），
// 命名空间隔离由下游侧的前缀配置承担。
const (
	RoleCodePlatformAdmin RoleCode = "platform_admin" // 内置「管理员」角色编码
	RoleCodeTenantAdmin   RoleCode = "tenant_admin"   // 内置「租户管理员」角色编码
)

// RoleCodePattern 角色编码形状：小写字母开头，仅小写字母/数字/下划线
// （与 AppCodePattern 同风格；编码会作为 claim 值跨系统传递，禁止大写与连字符）。
const RoleCodePattern = `^[a-z][a-z0-9_]*$`

var roleCodeRegexp = regexp.MustCompile(RoleCodePattern)

// IsValidRoleCode 校验角色编码形状（服务层入口白名单判据）。
func IsValidRoleCode(code RoleCode) bool {
	return roleCodeRegexp.MatchString(string(code))
}

// reservedRoleCodes 系统保留编码：这些都是下游**已供给策略**的契约值（下游按「前缀 + 编码」认策略名），
// 只允许种子/租户开通以 source=builtin 写入。若放开给自建角色占用，任何租户都能凭同码拿到该内置角色
// 对应的下游策略（跨租户提权）——下游前缀只能隔离命名空间，隔离不了同码，保留字是必须的第二道闸。
// 约定：凡为某编码在下游供给过策略，就必须在此登记。
var reservedRoleCodes = map[RoleCode]struct{}{
	RoleCodePlatformAdmin: {},
	RoleCodeTenantAdmin:   {},
}

// IsReservedRoleCode 判断编码是否为系统保留（仅内置角色可持有，自建角色不得占用）。
func IsReservedRoleCode(code RoleCode) bool {
	_, ok := reservedRoleCodes[code]
	return ok
}

// HasSystemAdmin 判断类型是否具备系统管理能力（仅管理员类型）。
func (t SysAdminType) HasSystemAdmin() bool {
	return t == SysAdminTypeAdmin
}

// IsBuiltinAdmin 判断角色是否为内置管理员（source=builtin 且 admin_type=admin）。
func (r *RoleEntity) IsBuiltinAdmin() bool {
	return r != nil && r.Source == RoleSourceBuiltin && r.AdminType == SysAdminTypeAdmin
}

type RoleEntity struct {
	gormdao.BaseEntity
	TenantID    string       `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID       string       `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id" json:"appID"`
	Name        string       `gorm:"column:name;type:varchar(128);not null;default:'';comment:角色名称" json:"name"`
	Code        RoleCode     `gorm:"column:code;type:varchar(64);not null;default:'';comment:角色编码(跨系统授权契约值,即 OIDC groups 取值)" json:"code"`
	Description string       `gorm:"column:description;type:varchar(256);not null;default:'';comment:角色描述" json:"description"`
	Source      RoleSource   `gorm:"column:source;type:varchar(16);not null;default:'custom';comment:角色来源(builtin/custom)" json:"source"`
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
