// Package contract 是 ark-iam 与 RP（业务应用）之间 OIDC wire 契约的唯一事实源。
//
// 签发侧（OP，apps/auth/internal/core/oidcop）与校验侧（RP，sdk/rp）共享本包，
// 使 claim 改名/改类型在编译期暴露，而不是上线后大面积 401。
//
// 依赖纪律：本包只允许标准库 + github.com/golang-jwt/jwt/v5。
package contract

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// TokenUsage 是 token 用途（私有 claim token_usage 的取值）。
// 空值表示自然人 token（非机器凭证）。
type TokenUsage string

const (
	// TokenUsageMachine 标识机器凭证签发的 token（API Key / client_credentials）。
	TokenUsageMachine TokenUsage = "machine"
)

// claim 名常量：签发与校验两侧共用，禁止在任何一侧裸写字面量。
const (
	// ClaimTokenUsage 是 token 用途 claim 名。
	ClaimTokenUsage = "token_usage"
	// ClaimTenantID 是租户 claim 名。
	ClaimTenantID = "tenant_id"
	// ClaimUserID 是用户 claim 名（值为 tenant_user.id，审计主键）。
	ClaimUserID = "user_id"
	// ClaimClientID 是 client id claim 名。
	ClaimClientID = "client_id"
	// ClaimSessionID 是中心会话标识 claim 名（OIDC 标准的 sid）。
	ClaimSessionID = "sid"
	// ClaimScope 是标准 OAuth scope 声明名（空格分隔）。
	ClaimScope = "scope"
	// ClaimActor 是代操作（actor）声明名，形如 act: {sub: <tenant_user.id>}。
	ClaimActor = "act"
	// ClaimPersonID 是自然人标识 claim 名（扩展声明；access token 默认集的一部分）。
	ClaimPersonID = "person_id"
)

// PersonSubjectPrefix 是 OIDC sub 中自然人标识的前缀（如 person:0198d5f6-xxxx）。
const PersonSubjectPrefix = "person:"

// BuildPersonSubject 把自然人 ID 组装为 OIDC sub（person:<id>）。
func BuildPersonSubject(personID string) string {
	return PersonSubjectPrefix + personID
}

// ParsePersonSubject 从 OIDC sub 解析自然人 ID；非 person 前缀或空 ID 返回 false。
func ParsePersonSubject(subject string) (string, bool) {
	if !strings.HasPrefix(subject, PersonSubjectPrefix) {
		return "", false
	}
	raw := strings.TrimPrefix(subject, PersonSubjectPrefix)
	if raw == "" {
		return "", false
	}
	return raw, true
}

// TokenClaims 是 OIDC access token 私有 claim 的单一事实源。
//
// 签发侧通过 OIDCPrivateClaims 产出 zitadel op.Storage 所需的扁平 map，
// 消费侧通过 jwt.ParseWithClaims 直接反序列化为该结构，两端共享同一份定义，
// 避免 claim 名/类型在 map 字面量与类型断言之间漂移。
// 自 string-id 改造起，TenantID/UserID 均为字符串主键（UUID v7）。
type TokenClaims struct {
	jwt.RegisteredClaims
	TokenUsage TokenUsage `json:"token_usage,omitempty"`
	TenantID   string     `json:"tenant_id,omitempty"`
	// UserID 是 tenant_user.id（租户成员主键），审计口径与 IAM 自身
	// created_by/updated_by 一致。人令牌由签发侧按 (tenant_id, person_id) 填，
	// 机器令牌为机器账号（tenant_user）主键。
	UserID   string `json:"user_id,omitempty"`
	ClientID string `json:"client_id,omitempty"`
	// SessionID 是中心会话标识（OIDC sid）：用于按会话作废与 back-channel 登出匹配。
	SessionID string `json:"sid,omitempty"`
	// PersonIDClaim 是自然人标识的显式声明（与 sub 的 person: 前缀同源，便于下游直读）。
	// 方法 PersonID() 优先返回它，缺失时回退解析 sub。
	PersonIDClaim string `json:"person_id,omitempty"`
	// Scope 是标准 OAuth scope 声明（空格分隔，RFC 6749 §3.3）。
	// 目录 API 等 M2M 权限判定据此进行。
	Scope string `json:"scope,omitempty"`
	// Actor 是代操作（actor）声明：机器凭证代表某人操作时承载原操作者，
	// wire 形态为 {"act":{"sub":"<tenant_user.id>"}}。
	Actor map[string]any `json:"act,omitempty"`
}

// OIDCPrivateClaims 把强类型编码为 op.Storage 需要的扁平 map（签发侧复用）。
func (c TokenClaims) OIDCPrivateClaims() map[string]any {
	m := make(map[string]any, 8)
	if c.TokenUsage != "" {
		m[ClaimTokenUsage] = c.TokenUsage
	}
	if c.TenantID != "" {
		m[ClaimTenantID] = c.TenantID
	}
	if c.UserID != "" {
		m[ClaimUserID] = c.UserID
	}
	if c.ClientID != "" {
		m[ClaimClientID] = c.ClientID
	}
	if c.SessionID != "" {
		m[ClaimSessionID] = c.SessionID
	}
	if c.PersonIDClaim != "" {
		m[ClaimPersonID] = c.PersonIDClaim
	}
	if c.Scope != "" {
		m[ClaimScope] = c.Scope
	}
	if c.Actor != nil {
		m[ClaimActor] = c.Actor
	}
	return m
}

// PersonID 从 Subject（形如 person:<id>）解析自然人 ID；
// 若 token 显式携带 person_id 声明则以它为准。
// 非 person 前缀（如机器凭证 sub=clientKeyPrefix）返回空字符串。
func (c *TokenClaims) PersonID() string {
	if c == nil {
		return ""
	}
	if c.PersonIDClaim != "" {
		return c.PersonIDClaim
	}
	pid, _ := ParsePersonSubject(c.Subject)
	return pid
}

// IsMachine 判断该 token 是否为机器凭证签发。
func (c *TokenClaims) IsMachine() bool {
	return c != nil && c.TokenUsage == TokenUsageMachine
}

// HasPerson 判断是否有自然人 sub（区别于机器凭证）。
func (c *TokenClaims) HasPerson() bool {
	return c != nil && strings.HasPrefix(c.Subject, PersonSubjectPrefix)
}
