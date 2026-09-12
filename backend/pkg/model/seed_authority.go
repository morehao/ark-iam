package model

// 种子字段权威矩阵（single writer per field）——本文件是「内置种子数据的哪个字段归谁写」的
// 唯一真相源，pkg/seed 按它决定收敛哪些字段，控制台服务按它拒写种子拥有的字段。
//
// 三条语义（与 docs/design/system-design.md §4.5 一致）：
//
//	reconcile   种子收敛：启动时必须与种子定义一致；写者唯一为种子，控制台对这些字段一律拒写
//	            （否则会出现"运维改完、重启被收回"的双写者）。
//	create_only 只播种：种子仅在行不存在时写入，此后永不回写；写者唯一为运维，控制台可改。
//	migrate_once 一次性迁移：跨版本的标识改名，以「当前值 == 历史种子值」为条件触发；
//	            迁移完成后自然失效，运维自定义值一律不动。迁移清单位于 pkg/seed。
//
// reconcile 的准入判据（2026-09-12 收窄，见 docs/design/system-design.md §4.5）：
// 只有「被控制台改写后会导致种子定位失效或鉴权被绕过」的字段才进 reconcile，共两类——
//   - 定位键：pkg/seed 靠它查行（tenant/application/application_client 的 code、
//     menu 的 app_id + code）。键一变，下次启动查不到该行 → 重复建一行，幂等性直接失效；
//   - 安全不变式：source（内置标记）、admin_type、平台租户 status。
// 不接受"产品身份/展示名"这类软理由：展示字段（应用名与描述、客户端名、菜单名/图标/排序/
// 可见性/路径/组件/层级）的跨版本自愈收益≈0，代价却是整页不可编辑，一律归运维（create_only）；
// 确需跨版本改名时按需登记 pkg/seed 的 seedMigrations（值匹配，运营改过就不动）。
//
// 硬规则：
//  1. 字段进入矩阵即表明其写者，未声明的字段视为 create_only（归运维），不得由种子回填；
//  2. reconcile 字段必须满足上面的准入判据，且必须在控制台侧拒写——只在一侧声明等于把
//     双写者的问题留到线上；
//  3. 历史改名（如 Default Tenant → 平台运营中心）用 migrate_once 表达，禁止用 reconcile
//     表达改名，否则运维的改名会在下次启动被无条件覆盖。

// SeedFieldMode 字段的写者语义（见文件头注释）。
type SeedFieldMode string

const (
	// SeedFieldReconcile 种子收敛：写者唯一为种子，控制台拒写。
	SeedFieldReconcile SeedFieldMode = "reconcile"
	// SeedFieldCreateOnly 只播种：写者唯一为运维，种子只在创建时写入。
	SeedFieldCreateOnly SeedFieldMode = "create_only"
	// SeedFieldMigrateOnce 一次性迁移：仅当归前值命中历史种子值时才改写。
	SeedFieldMigrateOnce SeedFieldMode = "migrate_once"
)

// 种子实体标识：矩阵行键，取业务领域名而非表名，避免与 TableName 常量耦合。
const (
	SeedEntityTenant            = "tenant"
	SeedEntityDepartment        = "department"
	SeedEntityApplication       = "application"
	SeedEntityApplicationClient = "application_client"
	SeedEntityMenu              = "menu"
	SeedEntityRole              = "role"
	SeedEntityPerson            = "person"
	SeedEntityUser              = "user"
)

// SeedPlatformTenantCode 平台自运营租户编码（种子身份）。
// pkg/seed 用它定位/创建平台租户，控制台用它判断"平台租户"并拒绝挂起。
const SeedPlatformTenantCode = "t_platform"

// 内置 OAuth 客户端编码（= OIDC client_id，种子身份）：pkg/seed 用它定位/播种/历史改名，
// platformadmin、tenantadmin 用它做令牌 audience 校验与 back-channel logout 的客户端识别。
// 与 SeedPlatformTenantCode 同理放在 pkg/model：种子、网关、前端共用同一个值，
// 改名时只需改这一处，不会出现"种子里改了、audience 没改"的错配。
// 取值遵循 ClientCodePattern（下划线连接，禁连字符）。
const (
	SeedBuiltinClientPlatformAdminWeb = "platform_admin_web"
	SeedBuiltinClientTenantAdminWeb   = "tenant_admin_web"
)

// SeedFieldAuthority 单条字段权威声明。
type SeedFieldAuthority struct {
	Entity string
	Field  string
	Mode   SeedFieldMode
}

// SeedFieldAuthorities 字段权威矩阵。
// 覆盖 pkg/seed 会写入或在控制台可编辑的字段；矩阵之外的字段视为 create_only。
var SeedFieldAuthorities = []SeedFieldAuthority{
	// 平台租户：code 与 name 只做一次性改名（旧库自愈），status 是安全性不变式（被挂起会整栈失联），
	// type/tag/db_user 属部署数据，种子的值只是"创建时的初值"。
	{SeedEntityTenant, "code", SeedFieldMigrateOnce},
	{SeedEntityTenant, "name", SeedFieldMigrateOnce},
	{SeedEntityTenant, "status", SeedFieldReconcile},
	{SeedEntityTenant, "type", SeedFieldCreateOnly},
	{SeedEntityTenant, "tag", SeedFieldCreateOnly},
	{SeedEntityTenant, "db_user", SeedFieldCreateOnly},
	// 平台租户根部门：与租户同名，仅在租户名发生一次性改名时同步一次（派生，不是每次启动跟随）。
	{SeedEntityDepartment, "name", SeedFieldMigrateOnce},
	{SeedEntityDepartment, "status", SeedFieldCreateOnly},
	{SeedEntityDepartment, "sort", SeedFieldCreateOnly},
	// 内置应用：source 是内置标记（安全不变式——平台侧"重置内置管理员口令"等能力依赖它定位内置对象），
	// 归种子收敛且控制台无写入入口。seed_key 是种子身份键（不可见不可写）：种子按它认行，
	// 因此 code 可归运维自由改名而不触发重建行。名称/描述属控制台展示身份（create_only）：种子的值
	// 只是创建时的初值，控制台改名/改描述后重启不回写。启停/排序/logo/主页同理。
	// code 另有一条 migrate_once：仅把历史连字符编码（platform-admin）值匹配改成下划线形态。
	{SeedEntityApplication, "seed_key", SeedFieldCreateOnly},
	{SeedEntityApplication, "code", SeedFieldMigrateOnce},
	{SeedEntityApplication, "source", SeedFieldReconcile},
	{SeedEntityApplication, "name", SeedFieldCreateOnly},
	{SeedEntityApplication, "description", SeedFieldCreateOnly},
	{SeedEntityApplication, "status", SeedFieldCreateOnly},
	{SeedEntityApplication, "sort", SeedFieldCreateOnly},
	{SeedEntityApplication, "logo_url", SeedFieldCreateOnly},
	{SeedEntityApplication, "homepage_url", SeedFieldCreateOnly},
	// 内置应用的 OAuth 客户端：code 是种子定位键（migrate_once 处理历史编码改名），app_id 是归属应用
	// ——平台管理后台客户端挂 platform_admin、租户管理后台客户端挂 tenant_admin，控制台不提供改归属的
	// 入口，故归种子收敛（历史版本把两个客户端都挂在 platform_admin，靠这条声明自愈）。
	// source 是内置标记（安全不变式）；name 属展示身份（create_only，归运维）；
	// 回调地址/授权类型/TTL 等与环境相关，归运维。
	{SeedEntityApplicationClient, "code", SeedFieldMigrateOnce},
	{SeedEntityApplicationClient, "app_id", SeedFieldReconcile},
	{SeedEntityApplicationClient, "source", SeedFieldReconcile},
	{SeedEntityApplicationClient, "name", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "redirect_uris", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "post_logout_redirect_uris", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "back_channel_logout_uri", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "grant_types", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "default_scopes", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "access_token_ttl", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "refresh_token_ttl", SeedFieldCreateOnly},
	// 菜单：**全部业务字段归运维（create_only）**，包括 app_id 与 code——种子靠 seed_key（种子身份键，
	// 不可见不可写）认行，不再靠 (app_id, code)，因此改编码/改归属都不会让种子重建行。
	// 种子的职责收敛为三件：按 seed_key 认行、缺失时创建、下线走 retiredMenus。
	// 代价（有意接受）：既有菜单行的跨版本结构变更不再自动生效，由运维在控制台完成。
	{SeedEntityMenu, "seed_key", SeedFieldCreateOnly},
	{SeedEntityMenu, "app_id", SeedFieldCreateOnly},
	{SeedEntityMenu, "code", SeedFieldCreateOnly},
	{SeedEntityMenu, "parent_id", SeedFieldCreateOnly},
	{SeedEntityMenu, "name", SeedFieldCreateOnly},
	{SeedEntityMenu, "path", SeedFieldCreateOnly},
	{SeedEntityMenu, "icon", SeedFieldCreateOnly},
	{SeedEntityMenu, "sort", SeedFieldCreateOnly},
	{SeedEntityMenu, "component", SeedFieldCreateOnly},
	{SeedEntityMenu, "type", SeedFieldCreateOnly},
	{SeedEntityMenu, "visibility", SeedFieldCreateOnly},
	{SeedEntityMenu, "status", SeedFieldCreateOnly},
	// 内置角色：admin_type 是系统管理能力的安全不变式（归种子），名称/描述归运维。
	{SeedEntityRole, "admin_type", SeedFieldReconcile},
	{SeedEntityRole, "name", SeedFieldCreateOnly},
	{SeedEntityRole, "description", SeedFieldCreateOnly},
	// 种子管理员：口令绝不覆盖（重置走专用接口），source 是内置标记（安全不变式）。
	{SeedEntityPerson, "password_encrypted", SeedFieldCreateOnly},
	{SeedEntityUser, "source", SeedFieldReconcile},
	{SeedEntityUser, "is_owner", SeedFieldCreateOnly},
}

// seedFieldModeIndex 矩阵的 (entity, field) → mode 索引（包初始化时构建，只读）。
var seedFieldModeIndex = func() map[[2]string]SeedFieldMode {
	index := make(map[[2]string]SeedFieldMode, len(SeedFieldAuthorities))
	for _, authority := range SeedFieldAuthorities {
		index[[2]string{authority.Entity, authority.Field}] = authority.Mode
	}
	return index
}()

// SeedFieldModeOf 返回字段的写者语义；未声明视为 SeedFieldCreateOnly（归运维，种子不回填）。
func SeedFieldModeOf(entity, field string) SeedFieldMode {
	if mode, ok := seedFieldModeIndex[[2]string{entity, field}]; ok {
		return mode
	}
	return SeedFieldCreateOnly
}

// SeedOwnsField 报告字段是否为种子收敛字段（reconcile）：
// 唯一写者 = 种子，控制台必须拒写。migrate_once 由迁移清单单独执行，不在此列。
func SeedOwnsField(entity, field string) bool {
	return SeedFieldModeOf(entity, field) == SeedFieldReconcile
}

// SeedReconcileFields 返回实体的种子收敛字段清单（pkg/seed 据此生成收敛更新集）。
func SeedReconcileFields(entity string) []string {
	fields := make([]string, 0, len(SeedFieldAuthorities))
	for _, authority := range SeedFieldAuthorities {
		if authority.Entity == entity && authority.Mode == SeedFieldReconcile {
			fields = append(fields, authority.Field)
		}
	}
	return fields
}
