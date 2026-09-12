package model

// 种子字段权威矩阵（single writer per field）——本文件是「内置种子数据的哪个字段归谁写」的
// 唯一真相源，pkg/seed 按它决定收敛哪些字段，控制台服务按它拒写种子拥有的字段。
//
// 三条语义（与 docs/design/seed-initialization-redesign-20260912.md 一致）：
//
//	reconcile   种子收敛：产品结构/身份的一部分，启动时必须与种子定义一致；写者唯一为种子，
//	            控制台对这些字段一律拒写（否则会出现"运维改完、重启被收回"的双写者）。
//	create_only 只播种：属部署/运营数据，种子仅在行不存在时写入，此后永不回写；写者唯一为运维。
//	migrate_once 一次性迁移：跨版本的核心标识改名，以「当前值 == 历史种子值」为条件触发；
//	            迁移完成后自然失效，运维自定义值一律不动。迁移清单位于 pkg/seed。
//
// 硬规则：
//  1. 字段进入矩阵即表明其写者，未声明的字段视为 create_only（归运维），不得由种子回填；
//  2. reconcile 字段必须同时在控制台侧拒写——只在一侧声明等于把双写者的问题留到线上；
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
	// 内置应用：source/name/description 是产品交付的身份，归种子；启停与排序归运维。
	{SeedEntityApplication, "code", SeedFieldMigrateOnce},
	{SeedEntityApplication, "source", SeedFieldReconcile},
	{SeedEntityApplication, "name", SeedFieldReconcile},
	{SeedEntityApplication, "description", SeedFieldReconcile},
	{SeedEntityApplication, "status", SeedFieldCreateOnly},
	{SeedEntityApplication, "sort", SeedFieldCreateOnly},
	{SeedEntityApplication, "logo_url", SeedFieldCreateOnly},
	{SeedEntityApplication, "homepage_url", SeedFieldCreateOnly},
	// 内置应用的 OAuth 客户端：回调地址/授权类型/TTL 与环境相关，归运维。
	{SeedEntityApplicationClient, "code", SeedFieldMigrateOnce},
	{SeedEntityApplicationClient, "source", SeedFieldReconcile},
	{SeedEntityApplicationClient, "name", SeedFieldReconcile},
	{SeedEntityApplicationClient, "redirect_uris", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "post_logout_redirect_uris", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "back_channel_logout_uri", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "grant_types", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "default_scopes", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "access_token_ttl", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "refresh_token_ttl", SeedFieldCreateOnly},
	// 菜单：结构（编码/路径/组件/类型/父子）与展示（名称/图标/排序/可见性）是控制台 IA，归种子；
	// 启停归运维。redirect/hidden/external_link/keep_alive 未在种子定义中出现，属 create_only。
	{SeedEntityMenu, "code", SeedFieldReconcile},
	{SeedEntityMenu, "parent_id", SeedFieldReconcile},
	{SeedEntityMenu, "name", SeedFieldReconcile},
	{SeedEntityMenu, "path", SeedFieldReconcile},
	{SeedEntityMenu, "icon", SeedFieldReconcile},
	{SeedEntityMenu, "sort", SeedFieldReconcile},
	{SeedEntityMenu, "component", SeedFieldReconcile},
	{SeedEntityMenu, "type", SeedFieldReconcile},
	{SeedEntityMenu, "visibility", SeedFieldReconcile},
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
