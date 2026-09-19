// Package middleware 承载 rpapi（面向应用的只读目录 API）自身的策略中间件。
//
// 与 auth 的 OIDC 中间件链**刻意分开**：这里只做 /v1/rp/* 的策略判定
// （aud 在 OIDCAuth 的 options 里收紧 + `directory.read` + tenant_id 前置校验），
// 避免"误复用 auth 的宽松链"导致目录越权（设计文档风险表已登记该风险）。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// DirectoryReadScope 是目录 API 要求的 scope（开放集合，只作策略与纵深防御，
// 真正的硬边界是服务端派生的 tenant_id 租户作用域）。
const DirectoryReadScope = "directory.read"

// RequireDirectoryRead 要求令牌携带 directory.read 且含 tenant_id。
//
// 状态码语义（设计文档关键决策六固定，SDK 侧据此分支：4xx 不重试、不降级）：
//   - 401：无/坏令牌（由上游 OIDCAuth 负责，这里兜底）；
//   - 403：缺 directory.read，或令牌没有 tenant_id（签发侧在 person→租户映射不唯一时
//     不写 tenant_id，此时必须 403，**不得**让 dbclient.ErrTenantScopeMissing 冒成 500）；
//   - 404/503：由控制器按查询结果给出。
func RequireDirectoryRead() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity := pkgmiddleware.OIDCIdentityFromContext(ctx)
		if identity == nil {
			// 未经过 OIDCAuth：绝不能放行，也不能当作"策略不满足"以外的语义。
			abort(ctx, http.StatusUnauthorized, code.RpDirectoryUnauthorizedError, code.GetError(code.RpDirectoryUnauthorizedError).Msg)
			return
		}
		if identity.TenantID == "" {
			abort(ctx, http.StatusForbidden, code.RpDirectoryForbiddenError, "令牌未携带租户作用域")
			return
		}
		// 租户作用域必须来自**服务端派生的** tenant_id：显式声明一次，
		// 后续 dao 查询按此隔离；不接受任何入参指定租户。
		gincontext.SetTenantScope(ctx, gcontext.CurrentScope(identity.TenantID))
		if !identity.HasScope(DirectoryReadScope) {
			abort(ctx, http.StatusForbidden, code.RpDirectoryForbiddenError, "缺少 "+DirectoryReadScope+" 权限")
			return
		}
		ctx.Next()
	}
}

// TenantIDFromContext 返回当前请求的租户 ID（已由 RequireDirectoryRead 校验非空）。
func TenantIDFromContext(ctx *gin.Context) string {
	return gincontext.GetTenantIDString(ctx)
}

// abort 以指定状态码 + 统一 error 信封结束请求。
func abort(ctx *gin.Context, status int, errorCode int, msg string) {
	ctx.AbortWithStatusJSON(status, gin.H{
		"code":      errorCode,
		"requestID": gincontext.GetRequestID(ctx),
		"msg":       msg,
		"data":      nil,
	})
}
