package contract

import "errors"

// 校验失败的显式错误分类。
//
// 消费者用 errors.Is 分支处理（不要解析错误字符串），
// 与主流实现同形态（如 zitadel 导出 ErrExpired/ErrSignatureUnsupportedAlg/ErrAzpInvalid）。
var (
	// ErrMalformedToken 表示令牌不是合法的 JWS（结构错误、编码错误）。
	ErrMalformedToken = errors.New("malformed token")
	// ErrUnsupportedAlg 表示签名算法不在允许列表内（如 HS256 混淆）。
	ErrUnsupportedAlg = errors.New("unsupported signing algorithm")
	// ErrUnknownKid 表示令牌头 kid 在密钥集里找不到（可能已轮换，需刷新 JWKS）。
	ErrUnknownKid = errors.New("unknown kid")
	// ErrMissingKid 表示令牌头缺少 kid（多 key 环境下无法定位验签密钥）。
	ErrMissingKid = errors.New("missing kid")
	// ErrBadSignature 表示签名验证失败。
	ErrBadSignature = errors.New("bad signature")
	// ErrExpired 表示令牌已过期。
	ErrExpired = errors.New("token expired")
	// ErrNotYetValid 表示令牌尚未生效（nbf 在未来）。
	ErrNotYetValid = errors.New("token not yet valid")
	// ErrBadIssuer 表示 iss 不匹配。
	ErrBadIssuer = errors.New("bad issuer")
	// ErrBadAudience 表示 aud 不匹配本应用。
	ErrBadAudience = errors.New("bad audience")
	// ErrMissingClaim 表示缺少必需的 claim（如既非 person 也非 machine 的令牌）。
	ErrMissingClaim = errors.New("missing required claim")
	// ErrUnauthorizedHeader 表示令牌头携带了被禁止的密钥分发参数
	// （jwk/jku/x5u，防 SSRF 与密钥替换）。
	ErrUnauthorizedHeader = errors.New("unauthorized token header")
	// ErrKeySourceUnavailable 表示密钥集不可用（首次拉取失败且无缓存）。
	ErrKeySourceUnavailable = errors.New("key source unavailable")
)
