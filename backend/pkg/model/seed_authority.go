package model

// 种子字段权威矩阵（single writer per field）——本文件是「内置数据的哪个字段归谁写」的
// 唯一真相源，控制台服务按它拒写不可变字段。
//
// 两条语义（与 docs/design/system-design.md §4.5 一致）：
//
//	immutable   L1 创建时写入，此后**必须保持不变**：控制台对这类字段一律拒写
//	            （否则会出现"运维改完、下次初始化被收回"的双写者，或鉴权被绕过）。
//	create_only L1 只在行不存在时写入，此后永不回写；写者唯一为运维，控制台可改。
//
// 为什么没有第三条：种子不再在启动期写入任何数据（L1 只在首次初始化执行一次，随后永久自锁），
// 因此**不再需要**跨版本的字段收敛（原 reconcile）与一次性改名（原 migrate_once）——
// 前者的执行体曾是 pkg/seed，后者的清单曾是 pkg/seed 的 seedMigrations，两者已删除。
// 内置数据的后续调整一律由运维在控制台完成（新增版本菜单见 `make print-builtin-menus`）。
//
// immutable 的准入判据：只有「被控制台改写后会导致鉴权被绕过或控制台自我锁死」的字段才进
// immutable，共三类——
//   - 安全不变式：source（内置标记）、admin_type（系统管理能力）、平台租户 status（挂起即整栈失联）；
//   - 身份编码：内置应用 code（各控制台菜单入口的定位值）、内置客户端 code（= client_id，
//     同时是网关 audience 白名单与前端构建期默认值）——改名会当场锁死对应控制台且界面无法自救；
//   - 客户端类型不变式：内置客户端的 token_endpoint_auth_method / require_pkce——它们必须保持
//     public + 强制 PKCE（浏览器客户端不可持密钥：RFC 6749 §10.1 / RFC 10017 §6.3.3.1）；
//     改成机密客户端会当场锁死控制台（前端不持密钥而 token 端点要求认证），关掉 PKCE 则去掉
//     公共客户端唯一的补偿控制（防降级）。
// 不接受"产品身份/展示名"这类软理由：展示字段（应用名与描述、客户端名、菜单名/图标/排序/
// 可见性/路径/组件/层级）一律归运维（create_only）；未声明的字段视为 create_only。
//
// 硬规则：
//  1. 字段进入矩阵即表明其写者；未声明的字段视为 create_only（归运维），种子不回填；
//  2. immutable 字段必须**在控制台侧有对应的拒写点**——只在一侧声明等于把双写者的问题留到线上；
//     矩阵登记的是"存在控制台写入入口的字段"，没有入口的字段（如 seed_key）不登记。
//  3. 展示类字段不接受"跨版本自动改名"的需求：确需改名由运维在控制台改，或按版本动作发版。

// SeedFieldMode 字段的写者语义（见文件头注释）。
type SeedFieldMode string

const (
	// SeedFieldImmutable 不可变：L1 创建时写入，之后控制台拒写。
	SeedFieldImmutable SeedFieldMode = "immutable"
	// SeedFieldCreateOnly 只播种：L1 只在创建时写入，之后写者唯一为运维，控制台可改。
	SeedFieldCreateOnly SeedFieldMode = "create_only"
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

// back-channel logout 接收端路径（相对 OIDC Issuer 的路径部分）。
//
// 三处必须一致，因此收口在这里：接收端注册（platformadmin/tenantadmin 的 app.go）、
// L1 播种时写入 application_client.back_channel_logout_uri 的派生结果。
// 只含路径不含主机：主机由 oidc.issuer 或 console 级覆盖决定（分体部署下两者可能不同）。
const (
	SeedBackChannelLogoutPathPlatform = "/bc-logout/platform"
	SeedBackChannelLogoutPathTenant   = "/bc-logout/tenant"
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
	// 平台租户：status 是安全不变式（被挂起会整栈失联，故控制台拒写）；name 归运维
	// （改成自己公司名后不被回写）；type/tag/db_user 属部署数据，种子的值只是"创建时的初值"。
	{SeedEntityTenant, "code", SeedFieldCreateOnly},
	{SeedEntityTenant, "name", SeedFieldCreateOnly},
	{SeedEntityTenant, "status", SeedFieldImmutable},
	{SeedEntityTenant, "type", SeedFieldCreateOnly},
	{SeedEntityTenant, "tag", SeedFieldCreateOnly},
	{SeedEntityTenant, "db_user", SeedFieldCreateOnly},
	// 平台租户根部门：创建时取租户名；之后按组织架构改名归运维。
	{SeedEntityDepartment, "name", SeedFieldCreateOnly},
	{SeedEntityDepartment, "status", SeedFieldCreateOnly},
	{SeedEntityDepartment, "sort", SeedFieldCreateOnly},
	// 内置应用：source 是内置标记（安全不变式——平台侧"重置内置管理员口令"等能力依赖它定位内置对象）；
	// code 是各控制台菜单入口的定位值（svcpermission.MyTree 按 platform_admin 查应用、
	// tenantadmin loadConsoleApps 只保留 tenant_admin），从控制台改名会当场锁死对应控制台且界面无法自救，
	// 故两者都 immutable。seed_key 是种子身份键（不可见不可写，无控制台入口，不登记）。
	// 名称/描述属控制台展示身份（create_only）：种子的值只是创建时的初值，控制台改名/改描述后不回写。
	{SeedEntityApplication, "code", SeedFieldImmutable},
	{SeedEntityApplication, "source", SeedFieldImmutable},
	{SeedEntityApplication, "name", SeedFieldCreateOnly},
	{SeedEntityApplication, "description", SeedFieldCreateOnly},
	{SeedEntityApplication, "status", SeedFieldCreateOnly},
	{SeedEntityApplication, "sort", SeedFieldCreateOnly},
	{SeedEntityApplication, "logo_url", SeedFieldCreateOnly},
	{SeedEntityApplication, "homepage_url", SeedFieldCreateOnly},
	// 应用角色模板：部署事实（该应用对外提供哪些跨系统契约值），归运维；种子只在创建时给初值，不回写。
	{SeedEntityApplication, "role_template", SeedFieldCreateOnly},
	// 内置应用的 OAuth 客户端：code（= client_id）同时是网关 aud 白名单、back-channel logout
	// 客户端识别与前端构建期 VITE_OIDC_CLIENT_ID 默认值的取值来源，改名会当场把对应控制台锁死，
	// 故 immutable；source 是内置标记（安全不变式），同 immutable。
	// app_id 是归属应用（创建时按控制台一一对应写入，无控制台改归属入口，不登记）；
	// name 属展示身份（create_only，归运维）；回调地址/授权类型/TTL 等与环境相关，归运维。
	{SeedEntityApplicationClient, "code", SeedFieldImmutable},
	{SeedEntityApplicationClient, "app_id", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "source", SeedFieldImmutable},
	// 认证方式与强制 PKCE 是浏览器公共客户端的安全不变式，同 immutable：
	// 内置两个控制台客户端是纯浏览器 SPA，其代码与配置会下发给每个用户，因此**必须**登记为
	// public（token_endpoint_auth_method=none）+ 强制 PKCE（require_pkce=enable）——
	// RFC 6749 §10.1 禁止为 user-agent 类客户端签发/要求客户端凭据，RFC 10017 §6.3.3.1
	// 要求浏览器客户端登记为 public 且授权服务器不得对其要求客户端认证；
	// 同时改写它们会当场把对应控制台锁死（前端不持密钥、token 端点要求认证 → invalid_client），
	// 且修复入口就在被锁死的控制台内部（与 code 同款自锁，无界面自救路径）。
	// require_pkce 被关掉则去掉公共客户端唯一的补偿控制（防降级）。
	{SeedEntityApplicationClient, "token_endpoint_auth_method", SeedFieldImmutable},
	{SeedEntityApplicationClient, "require_pkce", SeedFieldImmutable},
	{SeedEntityApplicationClient, "name", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "redirect_uris", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "post_logout_redirect_uris", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "back_channel_logout_uri", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "grant_types", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "default_scopes", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "access_token_ttl", SeedFieldCreateOnly},
	{SeedEntityApplicationClient, "refresh_token_ttl", SeedFieldCreateOnly},
	// 菜单：**全部业务字段归运维（create_only）**，包括 app_id 与 code——L1 靠 seed_key
	// （种子身份键，不可见不可写）认行，因此控制台改编码/改归属都不会影响后续行为。
	// L1 的职责收敛为两件：按 seed_key 认行、缺失时创建（下线与调整全在控制台完成）。
	// 代价（有意接受）：既有菜单行的跨版本结构变更不再自动生效，由运维按 `make print-builtin-menus`
	// 的清单在控制台完成。
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
	// 内置角色：admin_type 是系统管理能力的安全不变式（控制台拒写）。名称/描述/编码归"定义方"而非种子回写：
	// 编码是下游授权契约值（OIDC groups 取值），只有两个来源——产品锚点（种子定义，如 tenant_admin）
	// 与应用角色模板（application.role_template，开通应用时物化到各租户）；租户控制台对角色整体只读、
	// 没有写入入口，模板改名由 SyncAppRoleTemplateToTenants 同步（不是种子回写）。
	{SeedEntityRole, "admin_type", SeedFieldImmutable},
	{SeedEntityRole, "name", SeedFieldCreateOnly},
	{SeedEntityRole, "description", SeedFieldCreateOnly},
	{SeedEntityRole, "code", SeedFieldCreateOnly},
	// 种子管理员：口令绝不覆盖（重置走专用接口），source 是内置标记（安全不变式，控制台拒写）。
	{SeedEntityPerson, "password_encrypted", SeedFieldCreateOnly},
	{SeedEntityUser, "source", SeedFieldImmutable},
	{SeedEntityUser, "owner_type", SeedFieldCreateOnly},
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

// SeedFieldImmutable 报告字段是否不可变：L1 创建时写入，之后控制台必须拒写。
func SeedFieldIsImmutable(entity, field string) bool {
	return SeedFieldModeOf(entity, field) == SeedFieldImmutable
}

// SeedImmutableFields 返回实体的不可变字段清单（控制台据此实现/核对拒写点）。
func SeedImmutableFields(entity string) []string {
	fields := make([]string, 0, len(SeedFieldAuthorities))
	for _, authority := range SeedFieldAuthorities {
		if authority.Entity == entity && authority.Mode == SeedFieldImmutable {
			fields = append(fields, authority.Field)
		}
	}
	return fields
}
