// Package contract 的契约测试：claim 往返、空值省略、person: 前缀解析、
// logout_token 事件判定。这些断言是"签发侧与校验侧共享同一份定义"的安全网。
package contract

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOIDCPrivateClaims_RoundTrip(t *testing.T) {
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   BuildPersonSubject("person-1"),
			Audience:  jwt.ClaimStrings{"platform_admin_web"},
			Issuer:    "http://localhost:8099/oidc",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		TokenUsage:    TokenUsageMachine,
		TenantID:      "tenant-1",
		UserID:        "user-1",
		ClientID:      "client-1",
		SessionID:     "sid-1",
		PersonIDClaim: "person-1",
		Scope:         "openid profile",
		Actor:         map[string]any{"sub": "actor-1"},
	}
	flat := claims.OIDCPrivateClaims()
	require.Equal(t, TokenUsageMachine, flat[ClaimTokenUsage])
	require.Equal(t, "tenant-1", flat[ClaimTenantID])
	require.Equal(t, "user-1", flat[ClaimUserID])
	require.Equal(t, "client-1", flat[ClaimClientID])
	require.Equal(t, "sid-1", flat[ClaimSessionID])
	require.Equal(t, "person-1", flat[ClaimPersonID])
	require.Equal(t, "openid profile", flat[ClaimScope])
	require.Equal(t, map[string]any{"sub": "actor-1"}, flat[ClaimActor])
}

func TestOIDCPrivateClaims_OmitsEmpty(t *testing.T) {
	flat := TokenClaims{}.OIDCPrivateClaims()
	assert.Empty(t, flat, "空值必须省略而不是写入空串 claim")
}

func TestPersonID(t *testing.T) {
	t.Run("from subject prefix", func(t *testing.T) {
		c := &TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "person:88"}}
		assert.Equal(t, "88", c.PersonID())
	})
	t.Run("explicit claim wins", func(t *testing.T) {
		c := &TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "person:88"},
			PersonIDClaim:    "99",
		}
		assert.Equal(t, "99", c.PersonID())
	})
	t.Run("machine subject yields empty", func(t *testing.T) {
		c := &TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "ak_1234567"}}
		assert.Equal(t, "", c.PersonID())
		assert.False(t, c.HasPerson())
	})
	t.Run("nil safe", func(t *testing.T) {
		var c *TokenClaims
		assert.Equal(t, "", c.PersonID())
		assert.False(t, c.IsMachine())
	})
}

func TestTokenUsageMachine(t *testing.T) {
	assert.True(t, (&TokenClaims{TokenUsage: TokenUsageMachine}).IsMachine())
	assert.False(t, (&TokenClaims{}).IsMachine())
}

func TestParsePersonSubject(t *testing.T) {
	pid, ok := ParsePersonSubject("person:abc")
	assert.True(t, ok)
	assert.Equal(t, "abc", pid)

	_, ok = ParsePersonSubject("person:")
	assert.False(t, ok, "空 ID 必须判非法")

	_, ok = ParsePersonSubject("client-1")
	assert.False(t, ok)
}

// TestClaimsJSONWireFormat 锁定 wire 上的 claim 名与省略语义：
// 下游（含非 Go 实现）按这些键解析，改名属破坏性变更。
func TestClaimsJSONWireFormat(t *testing.T) {
	raw, err := json.Marshal(TokenClaims{
		TenantID: "t1",
		UserID:   "u1",
		ClientID: "c1",
	})
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.Equal(t, "t1", m["tenant_id"])
	assert.Equal(t, "u1", m["user_id"])
	assert.Equal(t, "c1", m["client_id"])
	// 未赋值的 claim 不出现在 wire 上
	for _, absent := range []string{"token_usage", "sid", "person_id", "act", "scope"} {
		_, ok := m[absent]
		assert.False(t, ok, "claim %s 不应出现在空值 token 上", absent)
	}
}

func TestLogoutTokenClaims_HasBackChannelLogoutEvent(t *testing.T) {
	c := &LogoutTokenClaims{Events: map[string]any{BackChannelLogoutEventURI: map[string]any{}}}
	assert.True(t, c.HasBackChannelLogoutEvent())
	assert.False(t, (&LogoutTokenClaims{}).HasBackChannelLogoutEvent())
	assert.False(t, (&LogoutTokenClaims{Events: map[string]any{"other": 1}}).HasBackChannelLogoutEvent())
	var nilClaims *LogoutTokenClaims
	assert.False(t, nilClaims.HasBackChannelLogoutEvent())
}
