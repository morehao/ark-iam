package internal

import (
	"context"

	"github.com/gin-gonic/gin"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/oidckit"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/glog"
)

// authMiddlewareForGroup 构造某个内置控制台应用（platform/tenant）版本组的鉴权中间件：
// OIDCAuth（验签 + iss/aud 收紧 + 可选 SSO 会话活性校验）+ RequireAppAccess（应用准入门禁）。
//
// 与现网三 app 的中间件装配逐段等价，配置来源为完整 pkgconfig.Config。
func (a *App) authMiddlewareForGroup(conf *pkgconfig.Config, audience string) []gin.HandlerFunc {
	issuer := ""
	if conf != nil {
		issuer = conf.OIDC.Issuer
	}
	// 解环接线：在本装配层 new 出 sdk/rp 的 Verifier 实现并注入 pkg 中间件
	// （pkg 只认 contract.Verifier 接口，不再 import sdk）。
	verifier := newVerifier(a.keySource, issuer, []string{audience})
	opts := []middleware.AuthOption{
		middleware.WithOIDCKeySource(a.keySource),
		middleware.WithVerifier(verifier),
	}
	if issuer != "" {
		opts = append(opts, middleware.WithOIDCIssuer(issuer))
	}
	opts = append(opts, middleware.WithOIDCAudiences(audience))
	if conf != nil && conf.OIDC.EnableSSOSessionValidation {
		ssoStore := sso.NewSSOSessionStore()
		opts = append(opts, middleware.WithOIDCSSOValidation(func(ctx context.Context, personID string, isMachineToken bool) bool {
			return validateSSOSession(ctx, ssoStore, personID, isMachineToken)
		}))
	}
	return []gin.HandlerFunc{
		middleware.OIDCAuth(opts...),
		middleware.RequireAppAccess(),
	}
}

// validateSSOSession 决定一次已验签请求是否仍有有效的 SSO 中心会话（按请求生效的撤销点）。
//
//   - 机器凭证：不依赖浏览器会话，直接放行；
//   - 未配置认证 Redis（dbclient.RedisCli == nil）：无法校验，放行（由 access token 短 TTL 兜底）；
//   - Redis 已配置但查询出错：fail-closed（返回 false）——否则已撤销的会话会继续用满令牌 TTL。
func validateSSOSession(ctx context.Context, store sso.SSOSessionStore, personID string, isMachineToken bool) bool {
	if isMachineToken {
		return true
	}
	if dbclient.RedisCli == nil {
		return true
	}
	active, err := store.HasActiveSession(ctx, personID)
	if err != nil {
		glog.Warnf(ctx, "[sdk/server] sso session validate fail, fail-closed, personID:%s, err:%v", personID, err)
		return false
	}
	return active
}

// registerBackChannelLogout 挂载 back-channel logout 接收端（/oidc/bc-logout/{platform|tenant}）。
func (a *App) registerBackChannelLogout(rg *gin.RouterGroup, conf *pkgconfig.Config, path, clientID string) {
	if conf == nil || a.keySource == nil {
		return
	}
	g := rg.Group("/oidc")
	g.Use(ginmiddleware.CORS())
	oidckit.RegisterReceiverRoutes(g, path, a.keySource, conf.OIDC.Issuer, clientID, nil)
}
