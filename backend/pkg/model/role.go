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
// 与 role.name 刻意分离：name 是中文可读展示、归运维改名；code 是契约值，改名即改变下游授权
// （下游缺策略时静默跳过，用户若再无其它策略即 Deny）。
//
// **取值只有两个来源，租户侧无写入口**（见 docs/design/system-design.md §5.4）：
//  1. 应用角色模板（application.role_template）：应用方（平台）定义契约值，开通应用时物化到各租户；
//  2. 产品锚点（下方内置编码）：随产品交付的内置角色，写死在代码里。
//
// 二者都是 `source=builtin`。租户自建角色（source=custom）的 code 恒为空串：它们只承载本系统内的
// 菜单权限，不参与跨系统契约——这正是"契约值不可能被租户改写/自造/撞码"的结构性保证。
type RoleCode string

// 内置角色编码取值（禁止硬编码）。这些是**产品锚点**：编码是制品事实，随产品交付，
// 由租户开通链路以 source=builtin 写入。取值刻意中性：不写下游产品语义（如 consoleAdmin），
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

// productAnchorRoleCodes 产品锚点编码集合：由本仓开通链路按常量写入的内置角色编码（制品事实）。
// 这些编码不允许出现在应用角色模板里——锚点角色由 `pkg/core/tenant` 按常量开通，若模板声明同码，
// 物化时会命中同一行并按模板改写锚点角色的名称，故在模板校验入口一律拒绝（svcapplication.buildRoleTemplate）。
var productAnchorRoleCodes = map[RoleCode]struct{}{
	RoleCodePlatformAdmin: {},
	RoleCodeTenantAdmin:   {},
}

// IsProductAnchorRoleCode 判断编码是否为产品锚点（模板不得声明、租户无写入口）。
func IsProductAnchorRoleCode(code RoleCode) bool {
	_, ok := productAnchorRoleCodes[code]
	return ok
}

// RoleTemplateItem 应用角色模板项：一个跨系统授权契约值（code）及其展示名（name）。
//
// 模板是**契约值的唯一来源**（产品锚点除外）：下游按「claim_prefix + code」认策略名，
// 同一个 code 在所有租户内共享同一条下游策略，因此 code 必须由应用方统一定义并保持稳定。
type RoleTemplateItem struct {
	Code RoleCode `json:"code"`
	Name string   `json:"name"`
}

// RoleTemplateItemList 应用角色模板项列表。
type RoleTemplateItemList []RoleTemplateItem

// Codes 取出模板中的契约值（按模板顺序；唯一性由服务层入口校验保证）。
func (l RoleTemplateItemList) Codes() []RoleCode {
	codes := make([]RoleCode, 0, len(l))
	for _, item := range l {
		codes = append(codes, item.Code)
	}
	return codes
}

// HasCode 判断模板是否声明了某契约值。
func (l RoleTemplateItemList) HasCode(code RoleCode) bool {
	for _, item := range l {
		if item.Code == code {
			return true
		}
	}
	return false
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
	TenantID string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id" json:"tenantID"`
	AppID    string `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id" json:"appID"`
	Name     string `gorm:"column:name;type:varchar(128);not null;default:'';comment:角色名称" json:"name"`
	// Code 角色编码：跨系统授权契约值（OIDC ID token `groups` 取值）。仅 source=builtin 的角色非空
	// ——取值来自应用角色模板（application.role_template）或产品锚点；租户自建角色恒为空串。
	Code        RoleCode     `gorm:"column:code;type:varchar(64);not null;default:'';comment:角色编码(跨系统授权契约值,即 OIDC groups 取值;仅模板/锚点角色非空)" json:"code"`
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
