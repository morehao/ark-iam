package ctroidc

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/morehao/ark-iam/pkg/identity"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/middleware"
	"github.com/morehao/ark-iam/auth/internal/service/svcoidc"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OIDCCtr struct {
	provider    *svcoidc.OIDCProvider
	oidcAuthSvc svcoidc.OIDCAuthSvc
	publicKey   *rsa.PublicKey
	// keySource 是本 OP 全部已发布公钥的进程内来源（含过渡期旧 key）：
	// auth 自身作为 RP 校验业务 token 时直接用它，零网络调用且自动跟随多 key 轮换。
	keySource rp.KeySource
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
	if provider != nil && provider.Storage != nil {
		if keys := provider.Storage.PublishedKeys(); len(keys) > 0 {
			ctr.keySource = rp.NewKeysFromSet(keys)
		}
	}
	return ctr
}

// KeySource 返回本 OP 已发布公钥的进程内来源（供鉴权中间件按 kid 取键）。
// 未装配成功（测试注入空 provider）时返回 nil，调用方应 fail-closed。
func (ctr *OIDCCtr) KeySource() rp.KeySource {
	if ctr == nil {
		return nil
	}
	return ctr.keySource
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
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.CompleteLogin(ctx.Request.Context(), &req)
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
	res, err := ctr.oidcAuthSvc.SelectTenant(ctx.Request.Context(), identity.OperatorFromGin(ctx), req.AuthRequestID, req.TenantID)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	setSSOSessionCookie(ctx, res.SessionID)
	gincontext.Success(ctx, res)
}

// ChangePassword 首次登录强制改密（临时密码）。
// @Tags OIDC
// @Summary 首次登录修改密码
// @Description 持临时密码（person.password_status = must_change）的自然人在登录被拦截后调用：校验当前密码与强度，
// @Description 设置新密码并清除强制改密标记，同时撤销该自然人既有会话；之后需重新登录。
// @accept application/json
// @Produce application/json
// @Param req body dtooidc.OIDCChangePasswordReq true "首次登录修改密码"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /oidc/login/changePassword [post]
func (ctr *OIDCCtr) ChangePassword(ctx *gin.Context) {
	var req dtooidc.OIDCChangePasswordReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.oidcAuthSvc.ChangePassword(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "密码修改成功")
}

func (ctr *OIDCCtr) RegisterPerson(ctx *gin.Context) {
	var req dtooidc.RegisterPersonReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.RegisterPerson(ctx.Request.Context(), &req)
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
	res, err := ctr.oidcAuthSvc.CreateTenant(ctx.Request.Context(), &req)
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
	res, err := ctr.oidcAuthSvc.LoginConfig(ctx.Request.Context(), req.AuthRequestID)
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

// noAppAccessHTML 无应用准入提示页（静默登录被拒时渲染，HTTP 403）。
// 纯静态文案：不回显 client_id / authRequestID 等参数，避免反射型注入。
const noAppAccessHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>无访问权限</title>
<style>
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;
background:#f5f6f8;color:#1f2329;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif}
.card{max-width:440px;padding:40px 36px;background:#fff;border-radius:12px;box-shadow:0 4px 16px rgba(0,0,0,.06);text-align:center}
h1{margin:0 0 12px;font-size:20px}
p{margin:0;font-size:14px;line-height:1.7;color:#646a73}
</style>
</head>
<body>
<div class="card">
<h1>你暂无该应用的访问权限</h1>
<p>你所在的租户未向你授予此应用的准入权限，请联系所在租户的管理员分配后再试。</p>
</div>
</body>
</html>`

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

	continueURL, err := ctr.oidcAuthSvc.CompleteLoginBySession(ctx.Request.Context(), identity.OperatorFromGin(ctx), authRequestID, sessionID)
	if err != nil {
		// 无应用准入（本次客户端应用下没有任何有准入的租户）：直接渲染提示页。
		// 不能走下面「清 cookie 回登录页」的分支——重新登录仍会得到同一结论，会形成登录死循环（AC-5）。
		if errors.Is(err, svcoidc.ErrSilentLoginNoAppAccess) {
			ctx.Header("Content-Type", "text/html; charset=utf-8")
			ctx.String(http.StatusForbidden, noAppAccessHTML)
			return
		}
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
		gincontext.WithRequestValue(ctx, oidcop.SSOSessionHintKey, sessionID)
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
