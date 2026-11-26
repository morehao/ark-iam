package appassembly

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth"
	"github.com/morehao/ark-iam/apps/platformadmin"
	"github.com/morehao/ark-iam/apps/tenantadmin"
	"github.com/morehao/ark-iam/contract"
	dirpkg "github.com/morehao/ark-iam/pkg/appassembly/directory"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/ark-iam/pkg/runtimesetting"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ErrDBDSNRequired 表示 Config.DB.DSN 缺失。
var ErrDBDSNRequired = errors.New("pkg/appassembly: config.DB.DSN is required")

// ErrAppAlreadyRunning 表示进程已持有一个 IAM 实例。
var ErrAppAlreadyRunning = errors.New("pkg/appassembly: an IAM application is already active in this process")

var appActive atomic.Bool

// App 是 pkg/appassembly 门面实例：持有 DB / Redis / 配置，按 Domains 编排 apps 各域库。
//
// 分层倒置下 App 不再持有业务实现（那些在 apps/* 库里），只做：
// 建连、KeySource 解析（同进程 auth 产出 / 独立进程远程 JWKS）、按域转发 Mount。
type App struct {
	ownsProcess      bool
	stopLogoutWorker func()
	closeKeys        func()
	closeOnce        sync.Once
	closeErr         error
	cfg              Config

	storeRaw *gorm.DB
	dbGetter gormdao.DBGetter
	redis    *redis.Client
	// setting 是运行时配置中心（golib gsetting，单进程共享一份），
	// 承载 notify / security.login 等分组；页面与业务消费者共用同一实例。
	setting *runtimesetting.Store
	// settingCache 是 gsetting 读路径缓存（redis / memory），Close 时释放其连接。
	settingCache gcache.Cache
	// keySource 是 OP 公钥来源，供 platform/tenant 域鉴权中间件验签。
	// 同进程（含 auth 域）由 auth 产出；独立进程由 cfg.KeySource/JWKSURL 远程解析。
	keySource oidckit.KeySource
}

// New 构造门面实例：校验配置并按需建立 DB/Redis 连接。
//
// New 只建立资源、不注册路由、不跑迁移；调用方随后显式调用
// RunMigrations / Mount / MountConfig。
func New(cfg Config) (*App, error) {
	if cfg.DB.DSN == "" {
		return nil, ErrDBDSNRequired
	}
	switch cfg.Process {
	case ProcessAll, ProcessAuth, ProcessPlatform, ProcessTenant:
	default:
		return nil, fmt.Errorf("pkg/appassembly: unknown process %q", cfg.Process)
	}
	// 域依赖校验（选定的域必须依赖闭合）。
	if err := validateDomainDeps(resolveDomains(cfg.Domains)); err != nil {
		return nil, err
	}
	// 非 dev 环境安全校验（fail-closed，须在建立连接前拒绝不安全配置）。
	if cfg.Raw != nil {
		if err := validateNonDevSecurityConfig(cfg.Raw, cfg.Env); err != nil {
			return nil, err
		}
	}

	if !appActive.CompareAndSwap(false, true) {
		return nil, ErrAppAlreadyRunning
	}
	app := &App{cfg: cfg, ownsProcess: true}
	ready := false
	defer func() {
		if !ready {
			_ = app.Close()
		}
	}()
	if err := app.initDB(); err != nil {
		return nil, err
	}
	if err := app.initRedis(); err != nil {
		return nil, err
	}
	if err := app.initSetting(); err != nil {
		return nil, err
	}
	dbclient.SetIamDB(app.storeRaw)
	dbclient.SetRedis(app.redis)
	runtimesetting.SetProcessStore(app.setting)
	ready = true
	return app, nil
}

// Mount 把选中的业务域挂到给定子路由组（不污染宿主根路由）。
//
// 路由布局（与现网一致，前缀 v1 由本方法内部补全）：
//
//	{Mount 传入的 rg}/v1/platform/*   platformadmin 域（menu/tenant/...）
//	{Mount 传入的 rg}/v1/tenant/*     tenantadmin 域
//	{Mount 传入的 rg}/v1/auth/*、/oidc、/install   auth 域
//
// 例：调用方 engine.Group("/iam").Mount 得到 /iam/v1/platform/menus。
func (a *App) Mount(rg *gin.RouterGroup) error {
	keep := make(map[Domain]bool)
	for _, d := range resolveDomains(a.cfg.Domains) {
		keep[d] = true
	}
	v1 := rg.Group("/v1")

	mountAuth := a.cfg.Process == ProcessAll || a.cfg.Process == ProcessAuth
	var platformKeys, tenantKeys []string
	if a.cfg.Process == ProcessAll || a.cfg.Process == ProcessPlatform {
		platformKeys = platformKeysFor(keep)
	}
	if a.cfg.Process == ProcessAll || a.cfg.Process == ProcessTenant {
		tenantKeys = tenantKeysFor(keep)
	}
	protected := (mountAuth && keep[DomainCoreAuth]) || len(platformKeys) > 0 ||
		len(tenantKeys) > 0 || (a.cfg.Process == ProcessAll && keep[DomainDirectory])
	if protected && a.cfg.VerifierBuilder == nil {
		return errors.New("pkg/appassembly: protected routes require an OIDC verifier builder")
	}

	// OP 先提供进程内公钥来源，分体管理进程才解析远程 JWKS。
	if mountAuth {
		a.mountAuth(v1.Group("/auth"), rg, keep)
	}
	if protected && a.keySource == nil {
		ks, err := a.resolveRemoteKeySource()
		if err != nil {
			return fmt.Errorf("pkg/appassembly: resolve OIDC key source: %w", err)
		}
		a.keySource = ks
		if a.cfg.KeySource == nil {
			if closer, ok := ks.(interface{ Close() }); ok {
				a.closeKeys = closer.Close
			}
		}
	}
	if protected && a.keySource == nil {
		return errors.New("pkg/appassembly: protected routes require an OIDC key source")
	}

	conf := a.rawConfig()
	if len(platformKeys) > 0 {
		platformGroup := v1.Group("/platform")
		platformGroup.Use(a.authMiddlewareForGroup(conf, model.SeedBuiltinClientPlatformAdminWeb)...)
		a.registerBackChannelLogout(rg, conf, model.SeedBackChannelLogoutPathPlatform, model.SeedBuiltinClientPlatformAdminWeb)
		platformadmin.Mount(platformGroup, a.dbGetter, platformKeys)
	}
	if len(tenantKeys) > 0 {
		tenantGroup := v1.Group("/tenant")
		tenantGroup.Use(a.authMiddlewareForGroup(conf, model.SeedBuiltinClientTenantAdminWeb)...)
		a.registerBackChannelLogout(rg, conf, model.SeedBackChannelLogoutPathTenant, model.SeedBuiltinClientTenantAdminWeb)
		tenantadmin.Mount(tenantGroup, a.dbGetter, tenantKeys)
	}
	return nil
}

// MountDirectory 挂载面向应用的只读目录 API（/v1/rp/directory/*，原 rpapi 并入）。
//
// 需在 Mount 之后调用（复用 Mount 解析出的 KeySource）。engine 必须是根引擎（目录按
// 服务标识段 "rp" 生成 /v1/rp/* 前缀，等价于重构前 gateway 的 rpapi.Init(engine,...)）。
func (a *App) MountDirectory(engine *gin.Engine) error {
	keep := make(map[Domain]bool)
	for _, d := range resolveDomains(a.cfg.Domains) {
		keep[d] = true
	}
	if a.cfg.Process != ProcessAll || !keep[DomainDirectory] {
		return nil
	}
	if a.keySource == nil || a.cfg.VerifierBuilder == nil {
		return errors.New("pkg/appassembly: mount directory requires OIDC key source and verifier builder")
	}
	conf := a.rawConfig()
	dirpkg.Mount(engine, conf, a.keySource, a.cfg.VerifierBuilder)
	return nil
}

// mountAuth 装配 auth 域：注入配置与 DB，注册业务路由与协议端点，返回 OP 进程内 KeySource。
func (a *App) mountAuth(authRG *gin.RouterGroup, engine *gin.RouterGroup, keep map[Domain]bool) {
	cfg := a.rawConfig()
	opts := auth.Options{
		OnLogoutWorkerStarted: func(stop func()) { a.stopLogoutWorker = stop },
		CoreAuth:              keep[DomainCoreAuth],
		Install:               keep[DomainInstall],
		Connector:             keep[DomainConnector],
		SelfService:           keep[DomainSelfService],
	}
	if !opts.CoreAuth && !opts.Install {
		return // 未选任何 auth 子域
	}
	a.keySource = auth.Mount(authRG, engine, a.dbGetter, cfg, opts, a.authVerifierBuilder)
}

// authVerifierBuilder 是注入 apps/auth 的验签器构造器：在本装配层用 Config.VerifierBuilder
// （由组合根注入的 sdk/rp 实现）构造 contract.Verifier，供 auth 业务路由中间件注入——apps
// 因此不必 import sdk，pkg 不 import sdk（消除反向依赖边）。
func (a *App) authVerifierBuilder(ks contract.KeySource, issuer string) contract.Verifier {
	return a.newVerifier(ks, issuer, nil)
}

// platformKeysFor 返回选中的平台管理端路由注册项。
func platformKeysFor(keep map[Domain]bool) []string {
	domainKey := map[Domain]string{
		DomainMenu:        "menu",
		DomainTenant:      "tenant",
		DomainApplication: "application",
		DomainOAuthClient: "oauth-client",
		DomainConnector:   "connector",
		DomainDomain:      "domain",
		DomainTenantApp:   "tenant-app",
		DomainAudit:       "audit",
		DomainSetting:     "setting",
	}
	return keysOf(keep, domainKey)
}

// tenantKeysFor 返回选中的租户管理端路由注册项。
func tenantKeysFor(keep map[Domain]bool) []string {
	domainKey := map[Domain]string{
		DomainUser:        "user",
		DomainDepartment:  "department",
		DomainRole:        "role",
		DomainMenu:        "menu",
		DomainMachineUser: "machine-user",
		DomainInvite:      "invite",
		DomainAPIKey:      "api-key",
		DomainApplication: "application",
	}
	return keysOf(keep, domainKey)
}

// keysOf 返回 keep 中命中的 key 列表。
func keysOf(keep map[Domain]bool, domainKey map[Domain]string) []string {
	out := make([]string, 0, len(domainKey))
	for d, key := range domainKey {
		if keep[d] {
			out = append(out, key)
		}
	}
	return out
}

// resolveRemoteKeySource 解析独立部署（非 auth 进程）的远程 JWKS 来源。
// 优先 cfg.KeySource；其次 cfg.JWKSURL / cfg.Issuer 经 oidckit 远程拉取。
func (a *App) resolveRemoteKeySource() (oidckit.KeySource, error) {
	if a.cfg.KeySource != nil {
		return a.cfg.KeySource, nil
	}
	conf := a.rawConfig()
	if conf == nil {
		conf = &pkgconfig.Config{}
	}
	local := *conf
	conf = &local
	// 用 Config 上的 JwksURL/Issuer 覆盖（支持不提供完整 Raw 的独立进程）。
	if a.cfg.JWKSURL != "" {
		conf.OIDC.JWKSURL = a.cfg.JWKSURL
	}
	if a.cfg.Issuer != "" && conf.OIDC.Issuer == "" {
		conf.OIDC.Issuer = a.cfg.Issuer
	}
	// 注入 sdk/rp 的具体 KeySource 构造器由组合根提供（pkg/oidckit 与 pkg/appassembly 均不
	// import sdk，解环）；未注入时相关分支返回 unavailable，中间件 fail-closed。
	return oidckit.NewKeySourceFromConfig(conf, a.cfg.KeysFromURL, a.cfg.KeysFromSet)
}

// DBGetter 返回实例持有的数据访问器。
func (a *App) DBGetter() gormdao.DBGetter {
	return a.dbGetter
}

// KeySource 返回 auth 域 OP 的进程内公钥来源（未装配 auth 且无远程来源时为 nil）。
func (a *App) KeySource() oidckit.KeySource {
	return a.keySource
}

// Redis 返回实例的 Redis 客户端；未配置时为 nil。
func (a *App) Redis() *redis.Client {
	return a.redis
}

// rawConfig 返回完整 pkgconfig.Config：优先 Config.Raw，否则合成最小配置（仅 dev）。
func (a *App) rawConfig() *pkgconfig.Config {
	if a.cfg.Raw != nil {
		return a.cfg.Raw
	}
	return a.synthPkgConfig()
}

// synthPkgConfig 基于精简字段合成一份最小 pkgconfig.Config（仅 dev 可用）。
func (a *App) synthPkgConfig() *pkgconfig.Config {
	env := a.cfg.Env
	if env == "" {
		env = "dev"
	}
	c := &pkgconfig.Config{}
	c.Server.Env = env
	c.OIDC.Issuer = a.cfg.Issuer
	return c
}
