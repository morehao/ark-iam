package contract

import (
	"context"
	"crypto/rsa"
)

// KeySource 是验签密钥来源：按 kid 返回 RSA 公钥。
//
// 这是 contract 层定义的最小验签协议——pkg 侧（pkg/oidckit、pkg/middleware）
// 只依赖本接口，具体实现（sdk/rp 的 *Keys）由装配层注入，从而斩断 pkg → sdk 的
// module 依赖边（见终局架构设计 §3.3 F1/F8）。
//
// 注意：本接口刻意使用 *rsa.PublicKey（stdlib crypto/rsa），不引入 jwt/jose，
// 与仓库现有 sdk/rp.KeySource 逐字一致，T2 仅需补编译期断言即可满足。
type KeySource interface {
	// PublicKey 返回 kid 对应的 RSA 公钥。
	// 找不到时报 ErrUnknownKid；密钥来源整体不可用时报 ErrKeySourceUnavailable。
	PublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

// Verifier 是框架无关的 access token 校验器。
//
// contract 定义接口、sdk/rp 提供实现、pkg/middleware 只依赖本接口——
// 与 KeySource 同理，用于反转 pkg → sdk 的依赖方向。
type Verifier interface {
	// Verify 校验原始令牌串并返回身份。
	// 失败时返回的错误可用 errors.Is 与 contract 的 sentinel 错误比较
	// （ErrExpired/ErrBadSignature/ErrBadIssuer/ErrBadAudience/ErrUnknownKid/
	// ErrUnsupportedAlg/ErrMalformedToken/ErrMissingClaim/ErrUnauthorizedHeader）。
	Verify(ctx context.Context, rawToken string) (*Identity, error)
}
