// Package app 是 tenantadmin 独立进程的装配入口（模板风格 Routers），由 cmd 调用、不产库。
//
// 为什么单独成包（而非 apps/tenantadmin 根包）：pkg/appassembly 会 import apps/tenantadmin
// （域库），若装配入口放进根包会形成 package import 环。独立为子包后依赖单向。
package app

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/tenantadmin"
	taconfig "github.com/morehao/ark-iam/apps/tenantadmin/config"
	"github.com/morehao/ark-iam/contract"
	"github.com/morehao/ark-iam/pkg/appassembly"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/seed"
	"github.com/morehao/ark-iam/sdk/rp"
)

// Routers 装配 tenantadmin 独立进程的全部路由（模板风格装配入口，cmd 只负责起进程）。
//
// 传入已 SetTrustedProxies 的 *gin.Engine；负责通用中间件、pkg/appassembly.New（域集/验签注入）、
// RunMigrations（if AutoMigrate）、Mount + MountConfig。返回 io.Closer 供进程退出释放资源。
//
// 独立部署无同进程 OP：tenant 域鉴权中间件的 KeySource 由 pkg/appassembly 按 Config.Issuer
// （= auth 的 issuer）远程拉 JWKS 解析。
func Routers(engine *gin.Engine) (io.Closer, error) {
	engine.Use(gin.Recovery())
	engine.Use(middleware.WithRequestFields())

	guardPrefixes := middleware.BootstrapGuardAllowPrefixes
	if taconfig.Conf.Server.Env == "dev" {
		guardPrefixes = append(append([]string{}, guardPrefixes...), "/tenantadmin")
	}
	engine.Use(middleware.BootstrapGuard(func(ctx context.Context) (bool, error) {
		return seed.IsInitialized(ctx, dbclient.IamDB(ctx))
	}, guardPrefixes...))

	cfg := appassembly.FromPkgConfig(taconfig.Conf, tenantDomains(), "", taconfig.Conf.OIDC.Issuer)
	cfg.Process = appassembly.ProcessTenant
	cfg.VerifierBuilder = verifierBuilder
	cfg.KeysFromURL = keysFromURL
	cfg.KeysFromSet = keysFromSet

	application, err := appassembly.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("[%s] appassembly.New fail: %w", tenantadmin.AppName, err)
	}
	if taconfig.Conf.DB.AutoMigrate {
		if err := application.RunMigrations(context.Background()); err != nil {
			_ = application.Close()
			return nil, fmt.Errorf("[%s] RunMigrations fail: %w", tenantadmin.AppName, err)
		}
	}
	if err := application.Mount(engine.Group("")); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("[%s] Mount fail: %w", tenantadmin.AppName, err)
	}
	if err := application.MountConfig(engine, ""); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("[%s] MountConfig fail: %w", tenantadmin.AppName, err)
	}
	return application, nil
}

// tenantDomains 返回 tenantadmin 独立进程的域集（user/department/role/menu/machine-user/
// invite/api-key/application）。role 依赖 menu，故含 menu。
func tenantDomains() []appassembly.Domain {
	return []appassembly.Domain{
		appassembly.DomainUser,
		appassembly.DomainDepartment,
		appassembly.DomainRole,
		appassembly.DomainMenu,
		appassembly.DomainMachineUser,
		appassembly.DomainInvite,
		appassembly.DomainAPIKey,
		appassembly.DomainApplication,
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
