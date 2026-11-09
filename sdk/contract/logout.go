package contract

import "github.com/golang-jwt/jwt/v5"

// BackChannelLogoutEventURI 是 logout_token 中 events 声明的标准事件 URI
// （OpenID Connect Back-Channel Logout 1.0 §2.4）。
const BackChannelLogoutEventURI = "http://schemas.openid.net/event/backchannel-logout"

// LogoutTokenClaims 是 OIDC Back-Channel Logout logout_token 的标准声明，
// 供 OP 签发侧与 RP 校验侧共享。
//
// 除 JWT 注册声明外，logout_token 必须包含：
//   - events: 包含 BackChannelLogoutEventURI 键
//   - jti:    唯一标识（防重放）
//   - iat:    签发时间（必填）
//   - sub:    用户标识（本仓 OP 始终下发；sid-only 属放宽项，见设计文档开放问题）
//   - sid:    会话 ID（可选，取决于 OP 是否支持 session 粒度）
//   - aud:    客户端 ID（必须是接收方 RP）
//   - nonce 必须不存在（keycloak 同款校验项）
type LogoutTokenClaims struct {
	jwt.RegisteredClaims
	SessionID string         `json:"sid,omitempty"`
	Events    map[string]any `json:"events,omitempty"`
	// Nonce 是 OIDC 标准声明；logout_token 中**必须不存在**（keycloak 同款校验项），
	// 因此这里显式解析它以便拒绝，而不是依赖"未知 claim 被忽略"。
	Nonce string `json:"nonce,omitempty"`
}

// HasBackChannelLogoutEvent 报告 logout_token 是否声明了 back-channel logout 事件。
func (c *LogoutTokenClaims) HasBackChannelLogoutEvent() bool {
	if c == nil || c.Events == nil {
		return false
	}
	_, ok := c.Events[BackChannelLogoutEventURI]
	return ok
}
