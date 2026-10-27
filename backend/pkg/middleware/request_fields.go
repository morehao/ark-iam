package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/gconstant"
)

// requestFieldKeyRequestID 是 glog 的 extra_keys 配置里声明的日志字段名。
// glog 用 ctx.Value(key) 取该字段（见 glog/driver/zap 的 extra fields 提取），
// 因此这个值必须落在**请求上下文**里，而不是只写在 gin Keys。
const requestFieldKeyRequestID = "requestID"

// contextKey 用于避免与其他包的同名 string key 冲突（context.WithValue 的 key
// 必须是可比较类型，这里用包内私有类型保证不串号）。
type contextKey struct{ name string }

var requestIDContextKey = contextKey{name: requestFieldKeyRequestID}

// WithRequestFields 把 gin Keys 里的请求级横切字段搬进**请求上下文**，
// 让 service / dao 在只拿到纯 context.Context 时仍能输出这些日志字段。
//
// 为什么需要它：2026-10-27 的边界调整后 controller 用 ctx.Request.Context() 交给 service，
// 而 requestID 只写在 gin Keys 上——纯 ctx 取不到，glog 的 extra_keys 会静默丢字段。
// 本中间件挂在鉴权/守卫之后（越早越好），把值投影到请求上下文的同名 string key 上。
//
// 取值优先级：请求上下文已存在同名值 → 不覆盖；否则取 gin Keys 的
// gcontext.KeyRequestID，再回落到 gconstant.KeyAppRequestID（老写法）。
func WithRequestFields() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request == nil {
			ctx.Next()
			return
		}
		reqCtx := ctx.Request.Context()
		if _, exists := reqCtx.Value(requestIDContextKey).(string); exists {
			ctx.Next()
			return
		}
		requestID := gincontext.GetRequestID(ctx)
		if requestID == "" {
			requestID = ctx.GetString(gconstant.KeyAppRequestID)
		}
		if requestID == "" {
			ctx.Next()
			return
		}
		//nolint:staticcheck // SA1029：这里刻意用 string key —— glog 按配置里的字符串名取值。
		ctx.Request = ctx.Request.WithContext(context.WithValue(reqCtx, requestFieldKeyRequestID, requestID))
		// 同时写入私有类型 key，供本包内需要类型安全的读者使用。
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), requestIDContextKey, requestID))
		ctx.Next()
	}
}
