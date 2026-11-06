package iam

import (
	"github.com/gin-gonic/gin"
	authconfig "github.com/morehao/ark-iam/iam/auth/config"
	authrouter "github.com/morehao/ark-iam/iam/auth/router"
	"github.com/morehao/ark-iam/iam/auth/service/svcdb"
	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

// mountAuth 装配 auth 域：注入配置与 DB，注册业务路由（authRG）与协议端点（engine），
// 返回 OP 的进程内 KeySource，供 platform/tenant 域鉴权中间件验签。
//
//	authRG —— 业务路由组（/v1/auth）
//	engine —— /oidc、/install 等协议端点挂载根
func (a *App) mountAuth(authRG *gin.RouterGroup, engine *gin.RouterGroup, keep map[Domain]bool) error {
	// auth 域深度依赖完整配置：优先用 Config.Raw，否则基于精简字段合成最小配置。
	cfg := a.cfg.Raw
	if cfg == nil {
		cfg = a.synthPkgConfig()
	}
	// 注入 auth 域运行时配置与共享 DBGetter（一进程一 IAM 实例语义）。
	authconfig.Set(cfg)
	svcdb.Set(a.dbGetter)

	opts := authrouter.Options{
		CoreAuth:    keep[DomainCoreAuth],
		Install:     keep[DomainInstall],
		Connector:   keep[DomainConnector],
		SelfService: keep[DomainSelfService],
	}
	if !opts.CoreAuth && !opts.Install {
		return nil // 未选任何 auth 子域
	}
	a.keySource = authrouter.Register(authRG, engine, a.dbGetter, opts)
	return nil
}

// synthPkgConfig 基于精简字段合成一份最小 pkgconfig.Config（仅 dev 可用）。
func (a *App) synthPkgConfig() *pkconfig.Config {
	env := a.cfg.Env
	if env == "" {
		env = "dev"
	}
	c := &pkconfig.Config{}
	c.Server.Env = env
	return c
}
