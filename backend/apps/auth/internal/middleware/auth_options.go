package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/golib/glog"
)

// OIDCBusinessAuthOptions 组装业务路由（/v1/auth/*）鉴权中间件选项：
// H3 iss 校验、skip 路径、SSO 会话活性校验（机器凭证直放行 / 无 Redis fail-open）。
func OIDCBusinessAuthOptions() []pkgmiddleware.AuthOption {
	opts := []pkgmiddleware.AuthOption{}
	if config.Conf != nil && config.Conf.OIDC.Issuer != "" {
		// H3：auth 作为 OP 自身，其 /v1/auth/* 接口接受任一合法 client 签发的 token，
		// 因此只校验 iss（必须是本 OP），aud 由各业务应用（RP）自行收紧。
		opts = append(opts, pkgmiddleware.WithOIDCIssuer(config.Conf.OIDC.Issuer))
	}
	ssoStore := sso.NewSSOSessionStore()
	opts = append(opts,
		pkgmiddleware.WithAuthSkipPaths(
			"/v1/auth/register",
			// 注意：必须与 router/auth.go 中注册的路由完全一致（复数 connectors）。
			// 第三方 IdP 回调（GET，无 Authorization 头）若被鉴权中间件拦截将直接 401，
			// 导致连接器登录不可用（曾有单复数拼写不一致的回归）。
			"/v1/auth/connectors/callback",
		),
		pkgmiddleware.WithOIDCSSOValidation(func(ctx *gin.Context, personID string, isMachineToken bool) bool {
			// 机器凭证（client_credentials/API Key）不依赖浏览器 SSO 会话活性，直接放行
			if isMachineToken {
				return true
			}
			// 无 Redis 时无法校验会话，采取放行（fail-open），避免破坏无 Redis 的环境
			if dbclient.RedisCli == nil {
				return true
			}
			active, err := ssoStore.HasActiveSession(ctx.Request.Context(), personID)
			if err != nil {
				glog.Warnf(ctx, "[middleware.OIDCBusinessAuthOptions] HasActiveSession fail, personID:%s, err:%v", personID, err)
				return true
			}
			return active
		}),
	)
	return opts
}
