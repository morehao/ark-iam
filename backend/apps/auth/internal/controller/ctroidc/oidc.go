package ctroidc

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OIDCCtr struct {
	oidcAuthSvc svcoidc.OIDCAuthSvc
}

func NewOIDCCtr(provider *svcoidc.OIDCProvider) *OIDCCtr {
	return &OIDCCtr{oidcAuthSvc: svcoidc.NewOIDCAuthSvc(provider)}
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
