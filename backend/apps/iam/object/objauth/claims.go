package objauth

import (
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// personSubjectPrefix 是 OIDC sub 中自然人标识的前缀（如 person:123）。
	personSubjectPrefix = "person:"

	// TokenUsageMachine 标识机器凭证签发的 token（API Key / client_credentials）。
	TokenUsageMachine = "machine"

	// claimTokenUsage 是 token 用途 claim 名。
	claimTokenUsage = "token_usage"
	// claimTenantID 是租户 claim 名。
	claimTenantID = "tenant_id"
	// claimUserID 是用户 claim 名。
	claimUserID = "user_id"
	// claimClientID 是 client id claim 名。
	claimClientID = "client_id"
)

// TokenClaims 是 OIDC access token 私密 claim 的单一事实源。
//
// 签发侧通过 OIDCPrivateClaims 产出 zitadel op.Storage 所需的扁平 map，
// 消费侧通过 jwt.ParseWithClaims 直接反序列化为该结构，两端共享同一份定义，
// 避免 claim 名/类型在 map 字面量与类型断言之间漂移。
type TokenClaims struct {
	jwt.RegisteredClaims
	TokenUsage string `json:"token_usage,omitempty"`
	TenantID   uint   `json:"tenant_id,omitempty"`
	UserID     uint   `json:"user_id,omitempty"`
	ClientID   string `json:"client_id,omitempty"`
}

// OIDCPrivateClaims 把强类型编码为 op.Storage 需要的扁平 map（签发侧复用）。
func (c TokenClaims) OIDCPrivateClaims() map[string]any {
	m := make(map[string]any, 4)
	if c.TokenUsage != "" {
		m[claimTokenUsage] = c.TokenUsage
	}
	if c.TenantID != 0 {
		m[claimTenantID] = c.TenantID
	}
	if c.UserID != 0 {
		m[claimUserID] = c.UserID
	}
	if c.ClientID != "" {
		m[claimClientID] = c.ClientID
	}
	return m
}

// PersonID 从 Subject（形如 person:<id>）解析自然人ID。
// 非 person 前缀（如机器凭证 sub=clientID）返回 0。
func (c *TokenClaims) PersonID() uint {
	if !strings.HasPrefix(c.Subject, personSubjectPrefix) {
		return 0
	}
	raw := strings.TrimPrefix(c.Subject, personSubjectPrefix)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// IsMachine 判断该 token 是否为机器凭证签发。
func (c *TokenClaims) IsMachine() bool {
	return c.TokenUsage == TokenUsageMachine
}

// HasPerson 判断是否有自然人 sub（区别于机器凭证）。
func (c *TokenClaims) HasPerson() bool {
	return strings.HasPrefix(c.Subject, personSubjectPrefix)
}
