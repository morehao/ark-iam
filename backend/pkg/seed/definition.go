package seed

import (
	"context"
	"errors"
	"strings"

	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/gorm"
)

// L1 首次引导的内置缺省值。
//
// 这些值同时是"未配置时的部署拓扑"，因此它们必须与改造前的硬编码完全一致：
// 运维不覆盖任何字段时，L1 的产物与旧的启动期播种逐字节相同。
const (
	defaultTenantName    = "平台运营中心"
	defaultAdminUsername = "admin"
	defaultAdminName     = "系统管理员"
	defaultAdminEmail    = "admin@example.com"
	defaultAdminPhone    = "13800000000"
	defaultIssuer        = "http://localhost:8100/oidc"

	defaultConsolePlatformRedirectURI   = "http://localhost:4001/auth/callback"
	defaultConsolePlatformPostLogoutURI = "http://localhost:4001/login"
	defaultConsoleTenantRedirectURI     = "http://localhost:4002/auth/callback"
	defaultConsoleTenantPostLogoutURI   = "http://localhost:4002/login"
)

// ErrAdminPasswordHashRequired 表示调用方未提供管理员口令摘要。
//
// pkg/seed 不再内置任何默认口令（历史实现里的 credential.BootstrapAdminPassword 已删除），
// 因此口令摘要必须由调用方（初始化接口）显式生成后传入；缺失即视为调用错误而非业务错误。
var ErrAdminPasswordHashRequired = errors.New("seed: AdminPasswordHash 必填，管理员口令摘要必须由调用方生成")

// Status 描述一次 Bootstrap 的落库结果。
type Status string

const (
	// StatusCreated 本次调用真正执行了 L1 引导（库从"未初始化"变为"已初始化"）。
	StatusCreated Status = "created"
	// StatusAlreadyInitialized 库已初始化（平台租户已存在），本次调用未写任何数据。
	StatusAlreadyInitialized Status = "already_initialized"
)

// Definition 是 L1 首次引导的输入：运维在初始化页面填写的身份信息 + 部署拓扑。
//
// 零值安全：除 AdminPasswordHash 外的字段留空即回落内置缺省值（见 withDefaults）。
type Definition struct {
	// TenantName 平台租户名称；根部门名与之同名。
	TenantName string
	// AdminUsername 内置管理员登录名（person.username）。
	AdminUsername string
	// AdminName 内置管理员显示名（person.name 与 user.name）。
	AdminName string
	// AdminEmail / AdminPhone 内置管理员的联系方式（person.primary_email / primary_phone）。
	AdminEmail string
	AdminPhone string
	// AdminPasswordHash 管理员口令的 bcrypt 摘要（必填）；明文口令不进入本包。
	AdminPasswordHash string
	// AdminPasswordStatus 管理员口令状态，零值回落 normal。
	AdminPasswordStatus model.PasswordStatus
	// Issuer OIDC Issuer，用于派生 back-channel logout 地址。
	Issuer string
	// Consoles 两个内置控制台的 OIDC 回调地址。
	Consoles ConsolesConfig
}

// ConsolesConfig 两个内置控制台的 OIDC 回调地址。
type ConsolesConfig struct {
	PlatformAdminWeb ConsoleRedirects
	TenantAdminWeb   ConsoleRedirects
}

// ConsoleRedirects 单个控制台的 OIDC 回调地址集合。
//
// 地址一律是**完整 URL**：不做 origin 派生——配置写什么就是什么。
// nil 切片回落内置缺省值，非 nil 的空切片按"显式配置为空"处理（不会回落）。
type ConsoleRedirects struct {
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	// BackChannelLogoutURI 为空时由 Issuer + model.SeedBackChannelLogoutPath* 派生；
	// 分体部署（auth 与 platformadmin/tenantadmin 不同主机）下单靠 Issuer 派生不出
	// 正确的接收端地址，此时显式配置本字段。
	BackChannelLogoutURI string
}

// defaultDefinition 返回全部走内置缺省值的定义（不含口令摘要）。
func defaultDefinition() Definition {
	return Definition{
		TenantName:    defaultTenantName,
		AdminUsername: defaultAdminUsername,
		AdminName:     defaultAdminName,
		AdminEmail:    defaultAdminEmail,
		AdminPhone:    defaultAdminPhone,
		Issuer:        defaultIssuer,
		Consoles: ConsolesConfig{
			PlatformAdminWeb: ConsoleRedirects{
				RedirectURIs:           []string{defaultConsolePlatformRedirectURI},
				PostLogoutRedirectURIs: []string{defaultConsolePlatformPostLogoutURI},
			},
			TenantAdminWeb: ConsoleRedirects{
				RedirectURIs:           []string{defaultConsoleTenantRedirectURI},
				PostLogoutRedirectURIs: []string{defaultConsoleTenantPostLogoutURI},
			},
		},
	}
}

// withDefaults 用内置缺省值补齐 Definition 的零值字段。
//
// 切片的判定分三种：nil → 回落缺省；非 nil → 原样保留（含空切片）；
// 元素为空字符串 → 回落缺省（避免配置里留了空串导致写入一个无效回调地址）。
func withDefaults(def Definition) Definition {
	fallback := defaultDefinition()

	if def.TenantName == "" {
		def.TenantName = fallback.TenantName
	}
	if def.AdminUsername == "" {
		def.AdminUsername = fallback.AdminUsername
	}
	if def.AdminName == "" {
		def.AdminName = fallback.AdminName
	}
	if def.AdminEmail == "" {
		def.AdminEmail = fallback.AdminEmail
	}
	if def.AdminPhone == "" {
		def.AdminPhone = fallback.AdminPhone
	}
	if def.AdminPasswordStatus == "" {
		def.AdminPasswordStatus = model.PasswordStatusNormal
	}
	if def.Issuer == "" {
		def.Issuer = fallback.Issuer
	}
	def.Consoles.PlatformAdminWeb = withConsoleDefaults(def.Consoles.PlatformAdminWeb, fallback.Consoles.PlatformAdminWeb)
	def.Consoles.TenantAdminWeb = withConsoleDefaults(def.Consoles.TenantAdminWeb, fallback.Consoles.TenantAdminWeb)
	// bc-logout 在补齐阶段就解析成最终值，让 Definition 完全自描述：
	// 调用方（初始化接口）回显给页面的地址与实际落库的地址因此不可能不一致。
	if def.Consoles.PlatformAdminWeb.BackChannelLogoutURI == "" {
		def.Consoles.PlatformAdminWeb.BackChannelLogoutURI = backChannelLogoutURI(def.Issuer, model.SeedBackChannelLogoutPathPlatform)
	}
	if def.Consoles.TenantAdminWeb.BackChannelLogoutURI == "" {
		def.Consoles.TenantAdminWeb.BackChannelLogoutURI = backChannelLogoutURI(def.Issuer, model.SeedBackChannelLogoutPathTenant)
	}
	return def
}

// withConsoleDefaults 补齐单个控制台的回调地址切片（nil 回落缺省，非 nil 原样保留）。
func withConsoleDefaults(cur, fallback ConsoleRedirects) ConsoleRedirects {
	if cur.RedirectURIs == nil {
		cur.RedirectURIs = fallback.RedirectURIs
	}
	if cur.PostLogoutRedirectURIs == nil {
		cur.PostLogoutRedirectURIs = fallback.PostLogoutRedirectURIs
	}
	return cur
}

// backChannelLogoutURI 由 Issuer 与接收端路径派生 back-channel logout 地址。
//
// Issuer 的尾斜杠与 path 的前导斜杠都做归一，避免出现 `//bc-logout` 或 `issuerbc-logout`；
// path 为空即返回归一后的 Issuer（调用方显式留空表示"不启用"时不得凭空拼出地址）。
func backChannelLogoutURI(issuer, path string) string {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	if path == "" {
		return issuer
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return issuer + path
}

// Bootstrap 执行 L1 首次引导：库未初始化时写入全部内置数据，已初始化时直接返回不写任何数据。
//
// 语义保证：
//   - 单事务：任一环节失败即整体回滚，不存在"只建了租户"之类的中间态；
//   - 自锁判定在事务内、advisory lock 之后，多副本并发只有一个能真正执行；
//   - 判定依据是平台租户行（库内事实），因此重启、换副本、清空审计日志都不会改变结论；
//   - 唯一写通道：这是本包对外的唯一写入入口（另两个公开函数只读）。
func Bootstrap(ctx context.Context, db *gorm.DB, def Definition) (Report, Status, error) {
	rep := Report{}
	if def.AdminPasswordHash == "" {
		return rep, "", ErrAdminPasswordHashRequired
	}
	def = withDefaults(def)

	status := StatusCreated
	txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSeed(tx); err != nil {
			return err
		}
		initialized, err := tenantInitialized(tx)
		if err != nil {
			return err
		}
		if initialized {
			status = StatusAlreadyInitialized
			return nil
		}
		return bootstrapAll(ctx, tx, &rep, def)
	})
	if txErr != nil {
		// 状态置空：调用方据此区分"已初始化"与"执行失败"，不会把失败误报成幂等命中。
		return rep, "", txErr
	}
	return rep, status, nil
}

// IsInitialized 报告租户数据是否已初始化（依据平台租户行是否存在）。
//
// 这是 /install 自锁与启动期守卫的唯一判定，也是幂等性的来源：判定不依赖任何进程内缓存，
// 因此重启、换副本、删审计日志都不会让它"忘记"。
func IsInitialized(ctx context.Context, db *gorm.DB) (bool, error) {
	return tenantInitialized(db.WithContext(ctx))
}

// tenantInitialized 查询平台租户是否存在（事务内/外通用）。
func tenantInitialized(db *gorm.DB) (bool, error) {
	entity, err := findTenantByCode(db, model.SeedPlatformTenantCode)
	if err != nil {
		return false, err
	}
	return entity != nil, nil
}

// SchemaReady 报告 schema 是否就绪（AutoMigrate 是否已建出核心表）。
//
// /install/status 用它区分两种"未初始化"：表还没建（启动配置或迁移有问题）
// 与表已建但数据未写入（正常待初始化状态）——两者的运维处置完全不同。
func SchemaReady(ctx context.Context, db *gorm.DB) (bool, error) {
	ready := db.WithContext(ctx).Migrator().HasTable(&model.TenantEntity{})
	return ready, nil
}
