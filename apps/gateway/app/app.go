// Package app 是 gateway 进程的装配入口（模板风格 Routers），由 cmd 调用、不产库。
//
// gateway 是纯聚合入口（不产库、绝不被 import），全量装配 auth/platformadmin/tenantadmin +
// 只读目录域。单独成包与其它 app 一致，避免与 cmd 混放。
package app

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	gwconfig "github.com/morehao/ark-iam/apps/gateway/config"
	"github.com/morehao/ark-iam/contract"
	"github.com/morehao/ark-iam/pkg/appassembly"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/seed"
	"github.com/morehao/ark-iam/sdk/rp"
)

// AppName 是 gateway 进程名（日志/健康检查/文档前缀）。
const AppName = "gateway"

// Routers 装配 gateway 全部路由（模板风格装配入口，cmd 只负责起进程）。
//
// 全量装配（Domains=nil → 全量域），行为与重构前等价：同一套路由布局、同一套鉴权中间件
// （OIDCAuth + RequireAppAccess）、同一套 back-channel logout；并额外挂只读目录 API。
// 同进程含 auth 域，产出进程内 KeySource（issuer 空），本地验签零网络。
func Routers(engine *gin.Engine) (io.Closer, error) {
	engine.Use(gin.Recovery())
	engine.Use(middleware.WithRequestFields())

	guardPrefixes := middleware.BootstrapGuardAllowPrefixes
	if gwconfig.Conf.Server.Env == "dev" {
		// dev 的 swagger/redoc 由 gindocs 挂在 /<appName>/...，放行以免"想查文档却被告知先初始化"。
		guardPrefixes = append(append([]string{}, guardPrefixes...), "/auth", "/platformadmin", "/tenantadmin")
	}
	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
	}, guardPrefixes...))

	cfg := appassembly.FromPkgConfig(gwconfig.Conf, nil /* 全量 */, "", "")
	cfg.VerifierBuilder = verifierBuilder
	cfg.KeysFromURL = keysFromURL
	cfg.KeysFromSet = keysFromSet

	application, err := appassembly.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("[%s] appassembly.New fail: %w", AppName, err)
	}
	if gwconfig.Conf.DB.AutoMigrate {
		if err := application.RunMigrations(context.Background()); err != nil {
			_ = application.Close()
			return nil, fmt.Errorf("[%s] RunMigrations fail: %w", AppName, err)
		}
	}
	// P0 多 issuer：按配置启用「Host→binding 解析」中间件（默认关，走静态 issuer）。
	// 必须在业务路由 Mount 之前注册，使每个请求先解析一次 Host 并写 pkg/oidcctx，
	// 供 /config.json、鉴权、/oidc/* 协议栈按请求 issuer 生效（见 pkg/appassembly/hostresolve.go）。
	if gwconfig.Conf.OIDC.EnableHostBinding {
		application.MountHostResolver(engine)
	}
	if err := application.Mount(engine.Group("")); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("[%s] Mount fail: %w", AppName, err)
	}
	if err := application.MountConfig(engine, ""); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("[%s] MountConfig fail: %w", AppName, err)
	}
	// 只读目录 API（原 rpapi）已并入 appassembly 的 DomainDirectory；全量挂载已含该域。
	// 此处显式挂目录路由，复用 Mount 解析出的进程内 KeySource。
	if err := application.MountDirectory(engine); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("[%s] MountDirectory fail: %w", AppName, err)
	}
	return application, nil
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
