package objauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestOIDCPrivateClaimsRoundTrip 锁死签发侧（map 产物）与消费侧（ParseWithClaims 反序列化）
// 的一致性：同一份 TokenClaims 经 OIDCPrivateClaims 产出的扁平 claim，
// 重新序列化签成 token 后，必须能被消费侧解析回等价的 TokenClaims。
func TestOIDCPrivateClaimsRoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		claims TokenClaims
	}{
		{
			name:   "person token with tenant",
			claims: TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "person:88"}, TenantID: 7},
		},
		{
			name:   "api key machine token",
			claims: TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "person:88"}, TenantID: 1, UserID: 7, TokenUsage: TokenUsageMachine},
		},
		{
			name:   "client credentials token",
			claims: TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "client-1"}, ClientID: "client-1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			priv, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("generate rsa key: %v", err)
			}

			// 签发侧：TokenClaims -> 扁平 map，作为 JWT 顶层 claims 合并后签名。
			// 用 jwt.MapClaims 原生扁平化，与 zitadel op.Storage 的产物形态一致。
			flat := tc.claims.OIDCPrivateClaims()
			var reg map[string]any
			regBytes, err := json.Marshal(tc.claims.RegisteredClaims)
			if err != nil {
				t.Fatalf("marshal registered claims: %v", err)
			}
			if err := json.Unmarshal(regBytes, &reg); err != nil {
				t.Fatalf("merge registered claims: %v", err)
			}
			for k, v := range reg {
				flat[k] = v
			}
			flat["iat"] = time.Now().Unix()
			flat["exp"] = time.Now().Add(time.Hour).Unix()

			signed := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(flat))
			tokenStr, err := signed.SignedString(priv)
			if err != nil {
				t.Fatalf("sign token: %v", err)
			}

			// 消费侧：ParseWithClaims 反序列化为 TokenClaims
			parsed := &TokenClaims{}
			_, err = jwt.ParseWithClaims(tokenStr, parsed, func(tk *jwt.Token) (interface{}, error) {
				return &priv.PublicKey, nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
			if err != nil {
				t.Fatalf("parse token: %v", err)
			}

			if parsed.Subject != tc.claims.Subject {
				t.Errorf("subject mismatch: got %q want %q", parsed.Subject, tc.claims.Subject)
			}
			if parsed.TenantID != tc.claims.TenantID {
				t.Errorf("tenant_id mismatch: got %d want %d", parsed.TenantID, tc.claims.TenantID)
			}
			if parsed.UserID != tc.claims.UserID {
				t.Errorf("user_id mismatch: got %d want %d", parsed.UserID, tc.claims.UserID)
			}
			if parsed.TokenUsage != tc.claims.TokenUsage {
				t.Errorf("token_usage mismatch: got %q want %q", parsed.TokenUsage, tc.claims.TokenUsage)
			}
			if parsed.ClientID != tc.claims.ClientID {
				t.Errorf("client_id mismatch: got %q want %q", parsed.ClientID, tc.claims.ClientID)
			}
		})
	}
}

// TestTokenClaimsHelpers 覆盖 PersonID / IsMachine / HasPerson。
func TestTokenClaimsHelpers(t *testing.T) {
	personTC := &TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "person:123"}}
	if got := personTC.PersonID(); got != 123 {
		t.Errorf("person PersonID got %d want 123", got)
	}
	if !personTC.HasPerson() {
		t.Error("person token should HasPerson")
	}
	if personTC.IsMachine() {
		t.Error("person token should not be machine")
	}

	machineTC := &TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "client-1"}, TokenUsage: TokenUsageMachine}
	if got := machineTC.PersonID(); got != 0 {
		t.Errorf("machine PersonID got %d want 0", got)
	}
	if machineTC.HasPerson() {
		t.Error("machine token should not HasPerson")
	}
	if !machineTC.IsMachine() {
		t.Error("machine token should be machine")
	}

	invalidSub := &TokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "not-a-person"}}
	if got := invalidSub.PersonID(); got != 0 {
		t.Errorf("invalid subject PersonID got %d want 0", got)
	}
}

// TestOIDCPrivateClaimsOmitEmpty 校验零值字段不进入产出 map。
func TestOIDCPrivateClaimsOmitEmpty(t *testing.T) {
	empty := (TokenClaims{}).OIDCPrivateClaims()
	if len(empty) != 0 {
		t.Errorf("empty claims should produce empty map, got %v", empty)
	}

	onlyTenant := TokenClaims{TenantID: 9}.OIDCPrivateClaims()
	if len(onlyTenant) != 1 || onlyTenant["tenant_id"] != uint(9) {
		t.Errorf("tenant-only claims got %v", onlyTenant)
	}
}
