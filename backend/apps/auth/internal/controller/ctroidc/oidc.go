package ctroidc

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OIDCCtr struct {
	provider    *svcoidc.OIDCProvider
	oidcAuthSvc svcoidc.OIDCAuthSvc
	publicKey   *rsa.PublicKey
}

// NewOIDCCtr 自装配 OIDC OP：provider（签名/加密密钥、协议态 storage）、
// back-channel logout 发送器、OIDC 鉴权服务与签名公钥缓存。
// 装配失败（如非 dev 环境密钥缺失）直接 panic，与旧 app 装配层行为一致。
func NewOIDCCtr() *OIDCCtr {
	provider, err := svcoidc.SetupOIDCProvider()
	if err != nil {
		panic(fmt.Sprintf("[ctroidc.NewOIDCCtr] SetupOIDCProvider fail, err:%v", err))
	}
	if err := provider.StartLogoutWorker(context.Background()); err != nil {
		panic(fmt.Sprintf("[ctroidc.NewOIDCCtr] StartLogoutWorker fail, err:%v", err))
	}
	return newOIDCCtr(provider)
}

// NewOIDCCtrWithProvider 测试专用：注入轻量/空 provider，跳过真实装配（密钥、worker）。
func NewOIDCCtrWithProvider(provider *svcoidc.OIDCProvider) *OIDCCtr {
	return newOIDCCtr(provider)
}

func newOIDCCtr(provider *svcoidc.OIDCProvider) *OIDCCtr {
	ctr := &OIDCCtr{provider: provider, oidcAuthSvc: svcoidc.NewOIDCAuthSvc(provider)}
	if pub, err := provider.PublicKey(); err == nil {
		ctr.publicKey = pub
	}
	return ctr
}

// PublicKey 返回 OP 签名公钥，供业务路由鉴权中间件校验本 OP 签发的 token。
// 未装配成功（测试注入空 provider）时返回 nil。
func (ctr *OIDCCtr) PublicKey() *rsa.PublicKey {
	if ctr == nil {
		return nil
	}
	return ctr.publicKey
}

// Provider 返回底层 OP provider，供路由层装配 OIDC 中间件使用。
func (ctr *OIDCCtr) Provider() *svcoidc.OIDCProvider {
	return ctr.provider
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

func (ctr *OIDCCtr) RegisterPerson(ctx *gin.Context) {
	var req dtooidc.RegisterPersonReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.RegisterPerson(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *OIDCCtr) CreateTenant(ctx *gin.Context) {
	var req dtooidc.CreateTenantReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.CreateTenant(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// LoginConfig 登录页前置策略查询：由 authRequestID 反查应用是否允许自助注册。
// @Tags OIDC
// @Summary 登录页前置策略查询
// @accept application/json
// @Produce application/json
// @Param req body dtooidc.OIDCLoginConfigReq true "登录页策略查询请求"
// @Success 200 {object} gincontext.DtoRender{data=dtooidc.OIDCLoginConfigResp}
// @Router /oidc/login-config [post]
func (ctr *OIDCCtr) LoginConfig(ctx *gin.Context) {
	var req dtooidc.OIDCLoginConfigReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.LoginConfig(ctx, req.AuthRequestID)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
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
// 无 id_token_hint 时，把 SSO cookie 会话 ID 放入 request context，
// 供 storage 回退按会话确定撤销目标（H15）。
func (ctr *OIDCCtr) EndSession(ctx *gin.Context) {
	if sessionID, err := ctx.Cookie(sso.SessionCookieName); err == nil && sessionID != "" {
		reqCtx := context.WithValue(ctx.Request.Context(), oidcop.SSOSessionHintKey, sessionID)
		ctx.Request = ctx.Request.WithContext(reqCtx)
	}
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
