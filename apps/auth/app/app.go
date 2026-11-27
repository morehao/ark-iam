// Package app 是 auth 独立进程的装配入口（模板风格 Routers），由 cmd 调用、不产库。
//
// 为什么单独成包（而非 apps/auth 根包）：pkg/appassembly 会 import apps/auth（域库），
// 若装配入口也放进 apps/auth 根包，则形成 apps/auth → pkg/appassembly → apps/auth 的
// package import 环。独立为子包 apps/auth/app 后依赖单向：app → appassembly → auth(域库)。
package app

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth"
	authconfig "github.com/morehao/ark-iam/apps/auth/config"
	"github.com/morehao/ark-iam/contract"
	"github.com/morehao/ark-iam/pkg/appassembly"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/seed"
	"github.com/morehao/ark-iam/sdk/rp"
)

// Routers 装配 auth 独立进程的全部路由（模板风格装配入口，cmd 只负责起进程）。
//
// 传入已 SetTrustedProxies 的 *gin.Engine；本函数负责：
//   - 通用中间件（gin.Recovery + WithRequestFields + BootstrapGuard 未初始化守卫）；
//   - pkg/appassembly.New（域集/验签实现注入）+ RunMigrations（if AutoMigrate）；
//   - Mount（业务/协议路由）+ MountConfig（前端运行时配置端点）。
//
// 返回的 io.Closer 在进程退出时释放 DB/Redis/setting（defer closer.Close()）。
//
// auth 进程用独立域集（业务登录 + 安装引导 + connector + 自助），并产出进程内 KeySource
// （issuer 空 → 同进程直接取 auth 产出的 key set，零网络）。
func Routers(engine *gin.Engine) (io.Closer, error) {
	engine.Use(gin.Recovery())
	// 把 requestID 等横切字段搬进请求上下文：service/dao 只拿纯 ctx，日志 extra_keys 才取得到 requestID。
	engine.Use(middleware.WithRequestFields())

	// 未初始化守卫：空库启动的业务端点变 409 + 安装引导错误码，放行 /install 与健康/发现端点。
	guardPrefixes := middleware.BootstrapGuardAllowPrefixes
	if authconfig.Conf.Server.Env == "dev" {
		// dev 的 swagger/redoc 由 gindocs 挂在 /<appName>/...，放行以免"想查文档却被告知先初始化"。
		guardPrefixes = append(append([]string{}, guardPrefixes...), "/auth")
	}
	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
	}, guardPrefixes...))

	cfg := appassembly.FromPkgConfig(authconfig.Conf, authDomains(), "", "")
	cfg.Process = appassembly.ProcessAuth
	cfg.VerifierBuilder = verifierBuilder
	cfg.KeysFromURL = keysFromURL
	cfg.KeysFromSet = keysFromSet

	application, err := appassembly.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("[%s] appassembly.New fail: %w", auth.AppName, err)
	}
	if authconfig.Conf.DB.AutoMigrate {
		if err := application.RunMigrations(context.Background()); err != nil {
			_ = application.Close()
			return nil, fmt.Errorf("[%s] RunMigrations fail: %w", auth.AppName, err)
		}
	}
	if err := application.Mount(engine.Group("")); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("[%s] Mount fail: %w", auth.AppName, err)
	}
	if err := application.MountConfig(engine, ""); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("[%s] MountConfig fail: %w", auth.AppName, err)
	}
	return application, nil
}

// authDomains 返回 auth 独立进程的域集：登录/会话、安装引导、connector、自助。
func authDomains() []appassembly.Domain {
	return []appassembly.Domain{
		appassembly.DomainCoreAuth,
		appassembly.DomainInstall,
		appassembly.DomainConnector,
		appassembly.DomainSelfService,
	}
}

// verifierBuilder 用 sdk/rp 构造 contract.Verifier（装配层注入，pkg 不 import sdk）。
func verifierBuilder(ks contract.KeySource, issuer string, audiences []string) contract.Verifier {
	var opts []rp.VerifierOption
	if issuer != "" {
		opts = append(opts, rp.WithIssuer(issuer))
	}
	if len(audiences) > 0 {
		opts = append(opts, rp.WithAudiences(audiences...))
	}
	return rp.NewVerifier(ks, opts...)
}

// keysFromURL 用 sdk/rp 从远程 JWKS URL 构造 KeySource。
func keysFromURL(ctx context.Context, url string) (contract.KeySource, error) {
	return rp.NewKeys(ctx, url)
}

// keysFromSet 用 sdk/rp 从本地公钥快照构造 KeySource。
func keysFromSet(keys map[string]*rsa.PublicKey) contract.KeySource {
	return rp.NewKeysFromSet(keys)
}
