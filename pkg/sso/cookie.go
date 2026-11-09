package sso

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SessionCookieName 是 auth（OP）SSO 中心会话在浏览器侧的 cookie 名。
// 业务应用（RP）作为无状态侧不读取该 cookie，仅 OP 自身与 /oidc 流程使用。
const SessionCookieName = "iam_sso_session"

// DefaultSameSite 是 SSO cookie 的默认 SameSite（Lax）：同站跳转可携带，
// 又不引入跨站请求携带的 CSRF 风险面。
var DefaultSameSite = http.SameSiteLaxMode

// SetSessionCookie 写入 SSO 会话 cookie。
// secure/sameSite 由调用方依据部署环境（HTTPS、跨站场景）传入：
// 生产环境应 secure=true，SameSite 视跨域拓扑取 Lax（同站）或 None+Secure（跨站）。
func SetSessionCookie(ctx *gin.Context, name, value string, maxAge int, domain string, secure bool, sameSite http.SameSite) {
	ctx.SetSameSite(sameSite)
	ctx.SetCookie(name, value, maxAge, "/", domain, secure, true)
}

// ClearSessionCookie 清除 SSO 会话 cookie（登出/失效时调用），参数语义同 SetSessionCookie。
func ClearSessionCookie(ctx *gin.Context, name, domain string, secure bool, sameSite http.SameSite) {
	SetSessionCookie(ctx, name, "", -1, domain, secure, sameSite)
}
