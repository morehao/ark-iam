// Package iam 是可嵌入的 IAM 核心库（v3 主线方案）。
//
// 对外契约（见 .dsh/docs/specs/2026-11-06-embeddable-iam-design.md）：
//
//	app, err := iam.New(iam.Config{...})   // 实例化，不再依赖包级全局
//	app.RunMigrations(ctx)                 // 建表（含撞名检测）
//	app.Mount(engine.Group("/iam"))        // 挂业务域到子路由组，不污染宿主根路由
//	app.MountFrontend(engine, "/iam")      // embed 登录页/管理台 + 运行时配置
//
// 两种形态共用同一核心库：
//   - 形态 1（gateway 独立部署）：Domains=nil（全量）+ 外置前端
//   - 形态 2（三方内嵌单体）：Domains=默认最小集 + MountFrontend（embed）
package iam

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ErrNotImplemented 表示某能力尚未在对应阶段实现（随 R2~R6 逐步替换）。
var ErrNotImplemented = errors.New("iam: not implemented yet")

// ErrDBDSNRequired 表示 Config.DB.DSN 缺失。
var ErrDBDSNRequired = errors.New("iam: config.DB.DSN is required")

// App 是一个可嵌入的 IAM 实例，持有 DB / Redis / 路由 / OP 等依赖（不再用包级全局）。
type App struct {
	cfg Config

	// storeRaw 是实例持有的底层 gorm 连接（用于 Close 释放）。
	storeRaw *gorm.DB
	// dbGetter 数据访问器：service 层经 dao.WithDBGetter(a.dbGetter) 注入，避免依赖全局 IamDB。
	dbGetter gormdao.DBGetter
	// redis 可空。
	redis *redis.Client
	// keySource 是 auth 域 OP 的进程内公钥来源，供 platform/tenant 域鉴权中间件验签。
	keySource oidckit.KeySource
}

// New 构造一个 IAM 实例：校验配置并按需建立 DB/Redis 连接。
//
// 说明：New 只建立资源、不注册路由、不跑迁移；调用方随后显式调用
// RunMigrations / Mount / MountFrontend。
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
	return app, nil
}

// Mount 把选中的业务域挂到给定子路由组（不污染宿主根路由）。
//
// 路由布局（与现网一致，前缀 v1 由本方法内部补全）：
//
//	{Mount 传入的 rg}/v1/platform/*   platformadmin 域（menu/tenant/...）
//	{Mount 传入的 rg}/v1/tenant/*     tenantadmin 域（R3）
//	{Mount 传入的 rg}/v1/auth/*、/oidc、/install   auth 域（R3）
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

	conf := a.cfg.Raw
	if conf == nil {
		conf = a.synthPkgConfig()
	}

	// platform 域：鉴权中间件（OIDCAuth + RequireAppAccess）+ back-channel logout。
	if a.anyDomain(keep, []Domain{DomainMenu, DomainTenant, DomainApplication, DomainOAuthClient, DomainDomain, DomainTenantApp, DomainAudit}) {
		platformGroup := v1.Group("/platform")
		if a.keySource != nil {
			platformGroup.Use(a.authMiddlewareForGroup(conf, model.SeedBuiltinClientPlatformAdminWeb, "platformadmin")...)
			a.registerBackChannelLogout(rg, conf, model.SeedBackChannelLogoutPathPlatform, model.SeedBuiltinClientPlatformAdminWeb)
		}
		a.mountPlatformAdmin(platformGroup, keep)
	}

	// tenant 域：鉴权中间件 + back-channel logout。
	if a.anyDomain(keep, []Domain{DomainUser, DomainDepartment, DomainRole, DomainMachineUser, DomainInvite, DomainAPIKey}) {
		tenantGroup := v1.Group("/tenant")
		if a.keySource != nil {
			tenantGroup.Use(a.authMiddlewareForGroup(conf, model.SeedBuiltinClientTenantAdminWeb, "tenantadmin")...)
			a.registerBackChannelLogout(rg, conf, model.SeedBackChannelLogoutPathTenant, model.SeedBuiltinClientTenantAdminWeb)
		}
		a.mountTenantAdmin(tenantGroup, keep)
	}
	return nil
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





// DBGetter 返回实例持有的数据访问器；供需要显式注入 DAO 的调用方使用。
func (a *App) DBGetter() gormdao.DBGetter {
	return a.dbGetter
}

// KeySource 返回 auth 域 OP 的进程内公钥来源（未装配 auth 时为 nil）。
func (a *App) KeySource() oidckit.KeySource {
	return a.keySource
}

// Redis 返回实例的 Redis 客户端；未配置时为 nil。
func (a *App) Redis() *redis.Client {
	return a.redis
}
