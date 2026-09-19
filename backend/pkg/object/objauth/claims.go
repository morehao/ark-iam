package objauth

import (
	"github.com/morehao/ark-iam/sdk/contract"
)

// 本文件是 SDK 契约在 IAM 内部的**类型别名门面**：签发侧（oidcop）与内置应用
// 的鉴权中间件都通过这里引用同一份定义，避免内外部两处独立演化。
//
// 契约本体在 github.com/morehao/ark-iam/sdk/contract（对外可复用的唯一真源）。
// 使用别名而非重新定义，保证 `objauth.TokenClaims` 与 `contract.TokenClaims`
// 完全同一类型（可互相赋值、可跨包传递）。

// TokenUsage token 用途（私有 claim token_usage 的取值）。
// 空值表示自然人 token（非机器凭证）。
type TokenUsage = contract.TokenUsage

const (
	// TokenUsageMachine 标识机器凭证签发的 token（API Key / client_credentials）。
	TokenUsageMachine = contract.TokenUsageMachine
)

const (
	// PersonSubjectPrefix 是 OIDC sub 中自然人标识的前缀（如 person:0198d5f6-xxxx）。
	PersonSubjectPrefix = contract.PersonSubjectPrefix

	// ClaimTokenUsage 是 token 用途 claim 名。
	ClaimTokenUsage = contract.ClaimTokenUsage
	// ClaimTenantID 是租户 claim 名。
	ClaimTenantID = contract.ClaimTenantID
	// ClaimUserID 是用户 claim 名。
	ClaimUserID = contract.ClaimUserID
	// ClaimClientID 是 client id claim 名。
	ClaimClientID = contract.ClaimClientID
	// ClaimSessionID 是中心会话标识 claim 名（OIDC sid）。
	ClaimSessionID = contract.ClaimSessionID
)

// TokenClaims 是 OIDC access token 私密 claim 的单一事实源（SDK 契约别名）。
//
// 签发侧通过 OIDCPrivateClaims 产出 zitadel op.Storage 所需的扁平 map，
// 消费侧通过 jwt.ParseWithClaims 直接反序列化为该结构，两端共享同一份定义，
// 避免 claim 名/类型在 map 字面量与类型断言之间漂移。
type TokenClaims = contract.TokenClaims
