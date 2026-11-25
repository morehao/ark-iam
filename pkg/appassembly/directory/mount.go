// Package directory 是 IAM 只读目录 API（/v1/rp/directory/*）的实现（原 apps/rpapi 并入）。
//
// 由原 rpapi 的 controller/service/middleware/router 迁移而来，DTO 用 contract/directory、
// 数据访问复用 pkg/dao（租户隔离按 ctx 作用域注入，缺失即 fail-closed）。
//
// 鉴权链是本域**自己的**一条：OIDCAuth（issuer + aud 收紧到本应用）→ RequireDirectoryRead
// （directory.read + tenant_id 前置校验并声明租户作用域）。刻意不复用 auth 的宽松链，
// 避免目录越权读。
package directory

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/contract"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// Mount 在 engine 上挂载 /v1/rp/directory/* 目录路由（稳定装配面，供 pkg/appassembly 调用）。
//
// keySource 由调用方给出（同进程 gateway 由 auth 注入进程内 key set；独立进程按配置自建）。
// buildVerifier 是注入的验签器构造器（真源 pkg/middleware.VerifierBuilder，由组合根用 sdk/rp
// 实现注入；本包不 import sdk）。拿不到 keySource 或 builder 时中间件对每个请求 fail-closed
// 返回 401，绝不挂裸路由。
func Mount(engine *gin.Engine, conf *pkgconfig.Config, keySource contract.KeySource, buildVerifier pkgmiddleware.VerifierBuilder) {
	oidcOpts := []pkgmiddleware.AuthOption{pkgmiddleware.WithOIDCKeySource(keySource)}
	if conf != nil {
		if conf.OIDC.Issuer != "" {
			oidcOpts = append(oidcOpts, pkgmiddleware.WithOIDCIssuer(conf.OIDC.Issuer))
		}
		if keySource != nil && buildVerifier != nil {
			var auds []string
			if len(conf.OIDC.Audiences) > 0 {
				auds = conf.OIDC.Audiences
			}
			oidcOpts = append(oidcOpts, pkgmiddleware.WithVerifier(buildVerifier(keySource, conf.OIDC.Issuer, auds)))
		}
		// aud 收紧：只接受 aud 指向本应用（rpapi/目录）自身的 M2M 令牌。
		if len(conf.OIDC.Audiences) > 0 {
			oidcOpts = append(oidcOpts, pkgmiddleware.WithOIDCAudiences(conf.OIDC.Audiences...))
		}
	}

	groups := ginserver.NewRouterGroups(engine, "rp", []ginserver.VersionGroup{{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			pkgmiddleware.OIDCAuth(oidcOpts...),
			RequireDirectoryRead(),
		},
	}})

	v1 := groups.MustGetGroup(ginserver.ApiVersionV1)
	ctr := NewDirectoryCtr()
	v1.GET("/directory/members", ctr.Members)
	v1.GET("/directory/members/:userID", ctr.Member)
	v1.GET("/directory/departments/tree", ctr.DepartmentTree)
	v1.GET("/directory/roles", ctr.Roles)
}
