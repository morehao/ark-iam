// Package identity 承载请求级身份的读写契约。
//
// 写侧由鉴权中间件（pkg/middleware 的 OIDC/API Key 通道）在验签通过后调用，
// 读侧供业务与策略层取值（如 apps/auth 的应用级入口门禁、apps/rpapi 的目录策略）。
// 把这对读写独立成包，是为了让消费方不必依赖「鉴权中间件包」——策略层回答的是
// "该身份是否有权做这件事"，与"请求是怎么被鉴权的"是两件事，依赖方向应为
// middleware → identity（写）与 consumer → identity（读），middleware 不在读取路径上。
//
// 注意：鉴权（验签/租户解析/会话活性）一律由 pkg/middleware 完成，本包不做任何
// 校验，只做上下文的取值搬运；调用方拿到空值/nil 时必须按 fail-closed 处理。
package identity

import (
	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/sdk/rp"
)

// 上下文键保持历史字面量不变（值即契约，改动会让按原始字符串读取的调用点静默失配）。
const (
	// contextKeyClientID 是 access token 中 client_id 声明在 gin 上下文里的键。
	// 它标识"本次请求是经由哪个 OIDC 客户端进来的"，是应用级入口策略
	// （pkg/core/application 的两个开关）唯一的解析入口。
	// golib 的 gcontext 没有对应常量，因此在本包内定义并配套 ClientIDFromContext 读取。
	contextKeyClientID = "oidcClientID"

	// contextKeyOIDCIdentity 是完整 rp.Identity（含 Scopes）在 gin 上下文里的键，
	// 供下游做策略判定（鉴权已完成，此处只回答"该身份是否有权做这件事"）。
	contextKeyOIDCIdentity = "oidcIdentity"
)

// SetClientID 写入本次请求的 OIDC client_id（由鉴权中间件调用）。
func SetClientID(ctx *gin.Context, clientID string) {
	if ctx == nil {
		return
	}
	ctx.Set(contextKeyClientID, clientID)
}

// SetOIDCIdentity 写入校验通过的完整身份（由鉴权中间件调用）。
func SetOIDCIdentity(ctx *gin.Context, ident *rp.Identity) {
	if ctx == nil {
		return
	}
	ctx.Set(contextKeyOIDCIdentity, ident)
}

// ClientIDFromContext 读取鉴权中间件注入的 client_id；未注入（如 skip path、
// API Key 通道）时返回空字符串，调用方应对空值 fail-closed。
func ClientIDFromContext(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.GetString(contextKeyClientID)
}

// OIDCIdentityFromContext 取出鉴权中间件写入的完整 identity（未经过 OIDC 鉴权时返回 nil）。
//
// 用途是**策略**判定（scope/客户端维度），不是鉴权：鉴权已由 OIDC 中间件完成，
// 因此调用方拿到 nil 时应按"策略不满足"处理，绝不回退成放行。
func OIDCIdentityFromContext(ctx *gin.Context) *rp.Identity {
	if ctx == nil {
		return nil
	}
	value, ok := ctx.Get(contextKeyOIDCIdentity)
	if !ok {
		return nil
	}
	ident, ok := value.(*rp.Identity)
	if !ok {
		return nil
	}
	return ident
}
