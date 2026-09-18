package model

import (
	"regexp"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameApplication = "application"

// AppCodePattern 应用编码规则：以小写字母开头，仅含小写字母、数字与下划线（如 platform_admin）。
// 种子内置应用（platform_admin / tenant_admin）与控制台新建应用共用同一口径，禁止连字符。
const AppCodePattern = `^[a-z][a-z0-9_]*$`

var appCodeRegexp = regexp.MustCompile(AppCodePattern)

// IsValidAppCode 判断应用编码是否符合 AppCodePattern。
func IsValidAppCode(code string) bool { return appCodeRegexp.MatchString(code) }

// AppSource 应用来源。
//
// builtin 是 first_party 中受保护的子集（内置应用同样由平台自有），枚举取最具体值：
// 平台随产品交付的控制台应用（平台管理后台、租户管理后台）都落 builtin；
// first_party 留给「平台自建但非内置」的应用（当前种子未产生该来源，控制台也不再创建）。
type AppSource string

// 应用来源取值（禁止硬编码）。
const (
	AppSourceBuiltin    AppSource = "builtin"     // 平台内置：种子播种/随产品交付，禁删，菜单归各自专属控制台
	AppSourceFirstParty AppSource = "first_party" // 平台自建但非内置：可删，菜单并入所在控制台
	AppSourceThirdParty AppSource = "third_party" // 第三方接入：外部/租户接入的应用
)

// IsBuiltin 判断是否为平台内置应用（禁删 + 菜单归专属控制台）。
func (s AppSource) IsBuiltin() bool { return s == AppSourceBuiltin }

// IsPlatformOwned 判断是否平台自有（内置或自建），供「自有 vs 外部」口径使用，避免调用方枚举三值。
func (s AppSource) IsPlatformOwned() bool { return s == AppSourceBuiltin || s == AppSourceFirstParty }

// AppStatus 应用启停状态（启停语义统一使用 enable/disable，见 docs/design/glossary.md「启停状态」）。
type AppStatus string

// 应用状态取值（禁止硬编码）。
const (
	AppStatusEnable  AppStatus = "enable"  // 启用
	AppStatusDisable AppStatus = "disable" // 停用
)

// AppPersonCreateTenantPolicy application.allow_person_create_tenant。
type AppPersonCreateTenantPolicy string

// 个人自助建租户开关取值（禁止硬编码）。
const (
	AppPersonCreateTenantPolicyEnable  AppPersonCreateTenantPolicy = "enable"
	AppPersonCreateTenantPolicyDisable AppPersonCreateTenantPolicy = "disable"
)

// AppJoinByInvitePolicy application.allow_join_by_invite。
type AppJoinByInvitePolicy string

// 邀请加入开关取值（禁止硬编码）。
const (
	AppJoinByInvitePolicyEnable  AppJoinByInvitePolicy = "enable"
	AppJoinByInvitePolicyDisable AppJoinByInvitePolicy = "disable"
)

type ApplicationEntity struct {
	gormdao.BaseEntity
	// Code 应用编码：业务标识，**控制台可改**（应用级唯一；改它不影响菜单/订阅/角色——那些都挂 app_id）。
	// 种子的认行依据是 SeedKey，因此改名不会导致种子重建应用行。
	Code string `gorm:"column:code;type:varchar(64);not null;default:'';uniqueIndex;comment:应用编码" json:"code"`
	// SeedKey 种子身份键：内置应用的稳定标识（= 种子定义时的 code），创建时写入后不再变化，
	// 控制台不可见也不可写。种子查行与租户开通（ProvisionTenantAdmin 定位 tenant_admin 应用）
	// 都以它为依据；控制台自建应用恒为空串（空串不进部分唯一索引）。
	SeedKey                 string                      `gorm:"column:seed_key;type:varchar(64);not null;default:'';comment:种子身份键(内置应用稳定标识,控制台不可见);uniqueIndex:uk_application_seed_key_active,where:deleted_at IS NULL AND seed_key <> ''" json:"-"`
	AllowPersonCreateTenant AppPersonCreateTenantPolicy `gorm:"column:allow_person_create_tenant;type:varchar(16);not null;default:'disable';comment:个人是否可自助创建租户(enable/disable)" json:"allowPersonCreateTenant"`
	AllowJoinByInvite       AppJoinByInvitePolicy       `gorm:"column:allow_join_by_invite;type:varchar(16);not null;default:'disable';comment:是否允许通过邀请加入租户(enable/disable)" json:"allowJoinByInvite"`
	// RoleTemplate 应用角色模板：本应用对外提供的跨系统授权契约值清单（code + 展示名）。
	//
	// 契约值的**唯一来源**（产品锚点除外）：下游按「claim_prefix + code」认策略名，策略在下游是
	// 全局命名实体、全租户共用一条，故 code 必须由应用方在这里定义一次；租户只能"授权"，不能造值。
	// 开通应用（tenant_application 落行）与模板变更时，模板按 (tenant_id, app_id, source=builtin, code)
	// 幂等物化为各租户的角色行（见 pkg/core/tenant.SyncAppRoleTemplate）。
	//
	// 归属：部署数据，由平台侧维护（seed 权威 create_only，控制台不写种子值）。
	RoleTemplate RoleTemplateItemList `gorm:"column:role_template;type:json;serializer:json;not null;default:('[]');comment:应用角色模板(跨系统授权契约值清单: code+name)" json:"roleTemplate"`
	Name         string               `gorm:"column:name;type:varchar(128);not null;default:'';comment:应用名称" json:"name"`
	Description  string               `gorm:"column:description;type:text;comment:应用描述" json:"description"`
	LogoURL      string               `gorm:"column:logo_url;type:varchar(2048);not null;default:'';comment:应用logo" json:"logoURL"`
	HomepageURL  string               `gorm:"column:homepage_url;type:varchar(2048);not null;default:'';comment:应用主页" json:"homepageURL"`
	Source       AppSource            `gorm:"column:source;type:varchar(32);not null;default:'third_party';comment:应用来源(builtin内置/first_party第一方/third_party第三方)" json:"source"`
	Status       AppStatus            `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态" json:"status"`
	Sort         int                  `gorm:"column:sort;type:int;not null;default:0;comment:排序" json:"sort"`
	CreatedBy    string               `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy    string               `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy    string               `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (ApplicationEntity) TableName() string { return TableNameApplication }

type ApplicationEntityList []ApplicationEntity

func (l ApplicationEntityList) ToMap() map[string]ApplicationEntity {
	m := make(map[string]ApplicationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
