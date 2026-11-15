package internal

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth"
	"github.com/morehao/ark-iam/apps/platformadmin"
	"github.com/morehao/ark-iam/apps/tenantadmin"
	"github.com/morehao/ark-iam/contract"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/notify"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/ark-iam/sdk/rp"
	dirpkg "github.com/morehao/ark-iam/sdk/server/internal/directory"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ErrDBDSNRequired 表示 Config.DB.DSN 缺失。
var ErrDBDSNRequired = errors.New("sdk/server: config.DB.DSN is required")

// App 是 sdk/server 门面实例：持有 DB / Redis / 配置，按 Domains 编排 apps 各域库。
//
// 分层倒置下 App 不再持有业务实现（那些在 apps/* 库里），只做：
// 建连、KeySource 解析（同进程 auth 产出 / 独立进程远程 JWKS）、按域转发 Mount。
type App struct {
	cfg Config

	storeRaw *gorm.DB
	dbGetter gormdao.DBGetter
	redis    *redis.Client
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

	app := &App{cfg: cfg}
	if err := app.initDB(); err != nil {
		return nil, err
	}
	if err := app.initRedis(); err != nil {
		return nil, err
	}
	if err := app.initNotify(); err != nil {
		return nil, err
	}
	return app, nil
}

// initNotify 按配置装配临时凭据投递通道（pkg/notify），并注册为进程级默认。
//
// 配置来自完整 pkgconfig.Config 的 notify 段（cfg.Raw）。未提供 Raw 或总闸关闭时，
// 装配一个「关闭态」Notifier（Send 恒 ErrNoChannel，业务侧回退现状）。总闸开但无渠道时
// FromConfig 报 ErrChannelLessConfig：dev 降级为关闭态并告警，prod fail-closed 拒绝启动。
func (a *App) initNotify() error {
	var ncfg notify.Config
	if a.cfg.Raw != nil {
		ncfg = notifyConfigFrom(a.cfg.Raw.Notify)
	}
	n, err := notify.FromConfig(ncfg)
	if err != nil {
		if errors.Is(err, notify.ErrChannelLessConfig) && a.cfg.Env != "prod" {
			glog.Warnf(context.Background(), "[sdk/server.initNotify] notify.enabled=true but no channel enabled; degrade to disabled (dev)")
			notify.SetDefault(notify.Disabled())
			return nil
		}
		return fmt.Errorf("sdk/server: init notify failed: %w", err)
	}
	notify.SetDefault(n)
	return nil
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

	// auth 域先装配：业务路由 /v1/auth，协议端点 /oidc、/install（均在本 rg 前缀下），
	// 并产生 OP 的进程内 KeySource，供 platform/tenant 域鉴权中间件复用。
	authGroup := v1.Group("/auth")
	if err := a.mountAuth(authGroup, rg, keep); err != nil {
		return err
	}

	// auth 未装配时，从 Config 解析 KeySource（独立部署的 platform/tenant 进程远程 JWKS）。
	if a.keySource == nil {
		ks, err := a.resolveRemoteKeySource()
		if err != nil {
			// fail-closed：拿不到 OP 公钥就绝不放行任何 token（中间件统一 401）。
			a.keySource = nil
		} else {
			a.keySource = ks
		}
	}

	conf := a.rawConfig()

	// platform 域：鉴权中间件（OIDCAuth + RequireAppAccess）+ back-channel logout。
	if a.anyDomain(keep, []Domain{DomainMenu, DomainTenant, DomainApplication, DomainOAuthClient, DomainDomain, DomainTenantApp, DomainAudit}) {
		platformGroup := v1.Group("/platform")
		if a.keySource != nil {
			platformGroup.Use(a.authMiddlewareForGroup(conf, model.SeedBuiltinClientPlatformAdminWeb)...)
			a.registerBackChannelLogout(rg, conf, model.SeedBackChannelLogoutPathPlatform, model.SeedBuiltinClientPlatformAdminWeb)
		}
		platformadmin.Mount(platformGroup, a.dbGetter, a.platformKeys(keep))
	}

	// tenant 域：鉴权中间件 + back-channel logout。
	if a.anyDomain(keep, []Domain{DomainUser, DomainDepartment, DomainRole, DomainMachineUser, DomainInvite, DomainAPIKey}) {
		tenantGroup := v1.Group("/tenant")
		if a.keySource != nil {
			tenantGroup.Use(a.authMiddlewareForGroup(conf, model.SeedBuiltinClientTenantAdminWeb)...)
			a.registerBackChannelLogout(rg, conf, model.SeedBackChannelLogoutPathTenant, model.SeedBuiltinClientTenantAdminWeb)
		}
		tenantadmin.Mount(tenantGroup, a.dbGetter, a.tenantKeys(keep))
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
	if !a.anyDomain(keep, []Domain{DomainDirectory}) {
		return nil
	}
	conf := a.rawConfig()
	dirpkg.Mount(engine, conf, a.keySource)
	return nil
}

// mountAuth 装配 auth 域：注入配置与 DB，注册业务路由与协议端点，返回 OP 进程内 KeySource。
func (a *App) mountAuth(authRG *gin.RouterGroup, engine *gin.RouterGroup, keep map[Domain]bool) error {
	cfg := a.rawConfig()
	opts := auth.Options{
		CoreAuth:    keep[DomainCoreAuth],
		Install:     keep[DomainInstall],
		Connector:   keep[DomainConnector],
		SelfService: keep[DomainSelfService],
	}
	if !opts.CoreAuth && !opts.Install {
		return nil // 未选任何 auth 子域
	}
	a.keySource = auth.Mount(authRG, engine, a.dbGetter, cfg, opts, authVerifierBuilder)
	return nil
}

// authVerifierBuilder 是注入 apps/auth 的验签器构造器：在本装配层（internal，唯一 import
// sdk/rp 之处）用 sdk/rp 构造 contract.Verifier，供 auth 业务路由中间件注入——apps 因此
// 不必 import sdk（消除 apps → sdk 依赖边）。
func authVerifierBuilder(ks contract.KeySource, issuer string) contract.Verifier {
	if ks == nil {
		return nil
	}
	vopts := []rp.VerifierOption{}
	if issuer != "" {
		vopts = append(vopts, rp.WithIssuer(issuer))
	}
	return rp.NewVerifier(ks, vopts...)
}

// platformKeys 把 keep 中的 platform 域映射为注册 key 列表；为空时返回 nil（=全量）。
func (a *App) platformKeys(keep map[Domain]bool) []string {
	domainKey := map[Domain]string{
		DomainMenu:        "menu",
		DomainTenant:      "tenant",
		DomainApplication: "application",
		DomainOAuthClient: "oauth-client",
		DomainDomain:      "domain",
		DomainTenantApp:   "tenant-app",
		DomainAudit:       "audit",
	}
	return keysOf(keep, domainKey)
}

// tenantKeys 把 keep 中的 tenant 域映射为注册 key 列表；为空时返回 nil（=全量）。
func (a *App) tenantKeys(keep map[Domain]bool) []string {
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

// keysOf 返回 keep 中命中的 key 列表；空则返回 nil（调用方 Mount 视为全量）。
func keysOf(keep map[Domain]bool, domainKey map[Domain]string) []string {
	out := make([]string, 0, len(domainKey))
	for d, key := range domainKey {
		if keep[d] {
			out = append(out, key)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// anyDomain 报告 keep 中是否包含 want 任一域。
func (a *App) anyDomain(keep map[Domain]bool, want []Domain) bool {
	for _, d := range want {
		if keep[d] {
			return true
		}
	}
	return false
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
	// 用 Config 上的 JwksURL/Issuer 覆盖（支持不提供完整 Raw 的独立进程）。
	if a.cfg.JWKSURL != "" {
		conf.OIDC.JWKSURL = a.cfg.JWKSURL
	}
	if a.cfg.Issuer != "" && conf.OIDC.Issuer == "" {
		conf.OIDC.Issuer = a.cfg.Issuer
	}
	// 注入 sdk/rp 的具体 KeySource 构造器（pkg/oidckit 自身不 import sdk，解环）。
	return oidckit.NewKeySourceFromConfig(
		conf,
		func(ctx context.Context, url string) (contract.KeySource, error) { return rp.NewKeys(ctx, url) },
		func(keys map[string]*rsa.PublicKey) contract.KeySource { return rp.NewKeysFromSet(keys) },
	)
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
	return c
}

var _ = middleware.OIDCAuth
var _ = ginmiddleware.CORS
var _ = context.Background
var _ = fmt.Sprintf
var _ = dbclient.IamDB
