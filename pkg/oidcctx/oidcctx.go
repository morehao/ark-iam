// Package oidcctx 承载「本次请求的 issuer」在 context 中的读写契约。
//
// 边界（p0-issuer-context-contract §2 / DG-ISSUER-1）：
//   - 只做上下文传递，**不做鉴权**；鉴权在入口中间件（解析 Host binding 后写入）。
//   - 纯 context，不依赖 gin / identity / gorm。
//   - **单一真源**：入口中间件解析一次写入本 ctx；Gin controller 把该 ctx 交给 service；
//     chi 的 IssuerFromRequest 从 r.Context() 读同一值；op.IssuerFromContext 只用于 provider 内部。
//   - Operator.Issuer 仅是审计副本，不参与验签/签发。
package oidcctx

import "context"

type issuerCtxKey struct{}

// WithIssuer 把本次请求的 issuer 写入 ctx。
func WithIssuer(ctx context.Context, issuer string) context.Context {
	if issuer == "" {
		return ctx
	}
	return context.WithValue(ctx, issuerCtxKey{}, issuer)
}

// IssuerFromContext 取出本次请求的 issuer；不存在返回 ("", false)。
// 缺失 issuer 时调用方必须 fail-closed，不得回退中央 issuer。
func IssuerFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(issuerCtxKey{}).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}
