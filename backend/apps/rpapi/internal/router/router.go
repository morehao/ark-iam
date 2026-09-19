// Package router 注册 rpapi 应用的路由。
//
// 服务标识段用 "rp"（与 auth 的 "auth" 同级），因此 /v1/rp/directory/* 遵循
// 「一 app 一前缀」约定，不需要任何例外（设计文档关键决策六）。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/config"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/ark-iam/rpapi/internal/controller/ctrdirectory"
	rpamiddleware "github.com/morehao/ark-iam/rpapi/internal/middleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

// RegisterRouter 装配 rpapi 全部路由。
//
// 鉴权链是 rpapi **自己的**一条：OIDCAuth（issuer + aud 收紧到本应用）
// → RequireDirectoryRead（directory.read + tenant_id 前置校验，并声明租户作用域）。
// 刻意不复用 auth 的宽松链：目录 API 的鉴权缺失会直接变成越权读。
//
// keySource 由调用方给出（app 层已按「注入优先、否则按配置自建」解析）：
// 拿不到时中间件对每个请求 fail-closed 返回 401，绝不静默放行（不挂裸路由）。
func RegisterRouter(engine *gin.Engine, conf *config.Config, keySource oidckit.KeySource) {
	registerRouter(engine, conf, keySource)
}

// registerRouter 装配路由的注入版入口：keySource 由调用方给出，
// 使单测能用进程内 key set 走与生产完全一致的验签链（离线、无 JWKS 网络依赖）。
func registerRouter(engine *gin.Engine, conf *config.Config, keySource oidckit.KeySource) {
	oidcOpts := []pkgmiddleware.AuthOption{pkgmiddleware.WithOIDCKeySource(keySource)}
	if conf != nil {
		if conf.OIDC.Issuer != "" {
			oidcOpts = append(oidcOpts, pkgmiddleware.WithOIDCIssuer(conf.OIDC.Issuer))
		}
		// aud 收紧：只接受 aud 指向本应用（rpapi）自身的 M2M 令牌——
		// 与 tenantadmin/platformadmin 收紧到内置控制台客户端同理。
		if len(conf.OIDC.Audiences) > 0 {
			oidcOpts = append(oidcOpts, pkgmiddleware.WithOIDCAudiences(conf.OIDC.Audiences...))
		}
	}

	groups := ginserver.NewRouterGroups(engine, "rp", []ginserver.VersionGroup{{
		Version: ginserver.ApiVersionV1,
		Middlewares: []gin.HandlerFunc{
			pkgmiddleware.OIDCAuth(oidcOpts...),
			rpamiddleware.RequireDirectoryRead(),
		},
	}})

	directoryRouter(groups)
}

func directoryRouter(groups *ginserver.RouterGroups) {
	v1 := groups.MustGetGroup(ginserver.ApiVersionV1)
	ctr := ctrdirectory.NewDirectoryCtr()
	v1.GET("/directory/members", ctr.Members)
	v1.GET("/directory/members/:userID", ctr.Member)
	v1.GET("/directory/departments/tree", ctr.DepartmentTree)
	v1.GET("/directory/roles", ctr.Roles)
}
