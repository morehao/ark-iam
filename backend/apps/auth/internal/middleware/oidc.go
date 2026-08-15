package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
)

// gin 上下文暂存键：OIDC hint 先经 c.Set 暂存在 gin 上下文（handler 链内可见），
// 再由 CarryOIDCHints 在进入 op.Provider（http.Handler）前统一搬运到 request context。
const (
	ginKeyTenantHint   = "iamTenantHint"
	ginKeyResourceHint = "iamResourceHint"
)

// TenantHint 读取 authorize 请求的 tenant query 参数并暂存到 gin 上下文，
// 供 op.Storage.CreateAuthRequest 经 req.Context() 读取（见 CarryOIDCHints）。
func TenantHint() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if t := ctx.Query("tenant"); t != "" {
			ctx.Set(ginKeyTenantHint, t)
		}
		ctx.Next()
	}
}

// ResourceHint 读取 token 端点的 RFC 8707 resource 参数并暂存到 gin 上下文，
// 供 op.Storage 决定 access token 的 aud（见 CarryOIDCHints）。
func ResourceHint() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if r := ctx.PostForm("resource"); r != "" {
			ctx.Set(ginKeyResourceHint, r)
		}
		ctx.Next()
	}
}

// CarryOIDCHints 把 gin 上下文中暂存的 OIDC hint 搬运到 request context。
// http.Handler（zitadel op.Provider）只能看到 *http.Request 的 context，
// gin.Context 不会跨过该边界，因此必须在透传前完成这次搬运；
// 这是全仓库唯一修改 c.Request 的地方，其余中间件一律只读。
func CarryOIDCHints(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()
	if t := ctx.GetString(ginKeyTenantHint); t != "" {
		reqCtx = context.WithValue(reqCtx, svcoidc.TenantHintKey, t)
	}
	if r := ctx.GetString(ginKeyResourceHint); r != "" {
		reqCtx = context.WithValue(reqCtx, svcoidc.ResourceHintKey, r)
	}
	ctx.Request = ctx.Request.WithContext(reqCtx)
}

// OIDCSilentAuth 组装 /authorize 的静默登录中间件（L1）：
// prompt=none 静默登录失败跳回 redirect_uri 前，校验其确为该 client 注册的回调地址，
// 杜绝利用未注册 redirect_uri 构造开放重定向。
func OIDCSilentAuth(provider *svcoidc.OIDCProvider, ssoSessionCookieName string) gin.HandlerFunc {
	ssoStore := sso.NewSSOSessionStore()
	return pkgmiddleware.SilentSSORequired(ssoSessionCookieName,
		pkgmiddleware.WithSessionValidator(func(ctx *gin.Context, sessionID string) error {
			_, err := ssoStore.ValidateSession(ctx.Request.Context(), sessionID)
			return err
		}),
		pkgmiddleware.WithRedirectURIVerifier(provider.RedirectURIVerifier()),
	)
}
