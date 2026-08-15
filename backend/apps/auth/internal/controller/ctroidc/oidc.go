package ctroidc

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OIDCCtr struct {
	provider    *svcoidc.OIDCProvider
	oidcAuthSvc svcoidc.OIDCAuthSvc
}

func NewOIDCCtr(provider *svcoidc.OIDCProvider) *OIDCCtr {
	return &OIDCCtr{provider: provider, oidcAuthSvc: svcoidc.NewOIDCAuthSvc(provider)}
}

func (ctr *OIDCCtr) Login(ctx *gin.Context) {
	var req dtooidc.OIDCLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.CompleteLogin(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	setSSOSessionCookie(ctx, res.SessionID)

	gincontext.Success(ctx, res)
}

func (ctr *OIDCCtr) SelectTenant(ctx *gin.Context) {
	var req dtooidc.OIDCSelectTenantReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.SelectTenant(ctx, req.AuthRequestID, req.TenantID)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	setSSOSessionCookie(ctx, res.SessionID)
	gincontext.Success(ctx, res)
}

// setSSOSessionCookie 写 SSO 会话 cookie，Secure/SameSite 由配置决定（L2）：
// 生产（HTTPS）应 cookieSecure=true；跨站场景 sameSite=none 且必须配合 Secure。
func setSSOSessionCookie(ctx *gin.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	ttl := 86400
	domain := ""
	secure := false
	sameSite := sso.DefaultSameSite
	if config.Conf != nil {
		ttl = config.Conf.OIDC.SessionTTL
		domain = config.Conf.OIDC.SSOCookieDomain()
		secure = config.Conf.OIDC.CookieSecure
		sameSite = config.Conf.OIDC.CookieSameSiteMode()
	}
	if ttl <= 0 {
		ttl = 86400
	}
	sso.SetSessionCookie(ctx, sso.SessionCookieName, sessionID, ttl, domain, secure, sameSite)
}

func (ctr *OIDCCtr) SSOLogin(ctx *gin.Context) {
	authRequestID := ctx.Query("authRequestID")
	if authRequestID == "" {
		ctx.Redirect(302, config.Conf.OIDC.FrontendLoginURL)
		return
	}

	sessionID, err := ctx.Cookie(sso.SessionCookieName)
	if err != nil {
		frontendURL := config.Conf.OIDC.FrontendLoginURL + "?authRequestID=" + url.QueryEscape(authRequestID)
		ctx.Redirect(302, frontendURL)
		return
	}

	continueURL, err := ctr.oidcAuthSvc.CompleteLoginBySession(ctx, authRequestID, sessionID)
	if err != nil {
		domain := ""
		secure := false
		sameSite := sso.DefaultSameSite
		if config.Conf != nil {
			domain = config.Conf.OIDC.SSOCookieDomain()
			secure = config.Conf.OIDC.CookieSecure
			sameSite = config.Conf.OIDC.CookieSameSiteMode()
		}
		sso.ClearSessionCookie(ctx, sso.SessionCookieName, domain, secure, sameSite)
		frontendURL := config.Conf.OIDC.FrontendLoginURL + "?authRequestID=" + url.QueryEscape(authRequestID)
		ctx.Redirect(302, frontendURL)
		return
	}

	ctx.Redirect(302, continueURL)
}

// ProviderHandler 返回 OIDC 协议端点（.well-known/openid-configuration、authorize、
// token、userinfo 等）的透传处理器，将请求原样转发给 zitadel op.Provider。
// 透传前先把 gin 上下文中暂存的 OIDC hint 搬运到 request context——
// 这是值跨过 gin → http.Handler 边界的唯一通道（见 middleware.CarryOIDCHints）。
// provider 未初始化（如单元测试中的空 provider）时返回 200 占位处理器保持兼容。
func (ctr *OIDCCtr) ProviderHandler() gin.HandlerFunc {
	if ctr.provider != nil && ctr.provider.Provider != nil {
		handler := http.StripPrefix("/oidc", ctr.provider.Provider)
		return func(ctx *gin.Context) {
			middleware.CarryOIDCHints(ctx)
			handler.ServeHTTP(ctx.Writer, ctx.Request)
		}
	}
	return func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	}
}

// EndSession 处理 RP-Initiated Logout（/oidc/end_session）：
// 先清除 SSO 中心会话 cookie（L2），再交给 provider 完成协议侧登出。
func (ctr *OIDCCtr) EndSession(ctx *gin.Context) {
	clearSSOCookie(ctx)
	ctr.ProviderHandler()(ctx)
}

// LoggedOut 处理登出完成回跳（DefaultLogoutRedirectURI=/oidc/logged-out）：
// 清除 SSO cookie 后跳回前端登录页。
func (ctr *OIDCCtr) LoggedOut(ctx *gin.Context) {
	clearSSOCookie(ctx)
	ctx.Redirect(302, config.Conf.OIDC.FrontendLoginURL)
}

// clearSSOCookie 依据配置清除 SSO 会话 cookie（Secure/SameSite 与写入时一致，L2）。
func clearSSOCookie(ctx *gin.Context) {
	domain := ""
	secure := false
	sameSite := sso.DefaultSameSite
	if config.Conf != nil {
		domain = config.Conf.OIDC.SSOCookieDomain()
		secure = config.Conf.OIDC.CookieSecure
		sameSite = config.Conf.OIDC.CookieSameSiteMode()
	}
	sso.ClearSessionCookie(ctx, sso.SessionCookieName, domain, secure, sameSite)
}
