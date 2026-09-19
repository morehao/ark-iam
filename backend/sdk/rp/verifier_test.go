package rp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morehao/ark-iam/sdk/contract"
)

const (
	testIssuer   = "http://localhost:8099/oidc"
	testAudience = "platform_admin_web"
)

// testKey 是一把测试密钥及其 kid。
type testKey struct {
	kid string
	key *rsa.PrivateKey
}

func newTestKey(t *testing.T, kid string) testKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return testKey{kid: kid, key: key}
}

// jwksBody 把密钥列表编码为标准 JWKS 文档。
func jwksBody(t *testing.T, keys ...testKey) []byte {
	t.Helper()
	type jwkDoc struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	doc := struct {
		Keys []jwkDoc `json:"keys"`
	}{}
	for _, k := range keys {
		doc.Keys = append(doc.Keys, jwkDoc{
			Kty: "RSA",
			Kid: k.kid,
			Alg: "RS256",
			Use: "sig",
			N:   base64.RawURLEncoding.EncodeToString(k.key.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.key.E)).Bytes()),
		})
	}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	return raw
}

// jwksServer 起一个返回给定 JWKS 的测试服务器，并统计请求次数。
func jwksServer(t *testing.T, body func() []byte, requests *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests != nil {
			atomic.AddInt32(requests, 1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body())
	}))
	t.Cleanup(srv.Close)
	return srv
}

// signToken 用指定密钥签发 access token。
func signToken(t *testing.T, k testKey, mutate func(jwt.MapClaims)) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":       "person:person-1",
		"tenant_id": "tenant-1",
		"user_id":   "user-1",
		"client_id": testAudience,
		"iss":       testIssuer,
		"aud":       testAudience,
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	if mutate != nil {
		mutate(claims)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = k.kid
	signed, err := token.SignedString(k.key)
	require.NoError(t, err)
	return signed
}

func newTestVerifier(t *testing.T, ks KeySource, opts ...VerifierOption) *Verifier {
	t.Helper()
	base := []VerifierOption{
		WithKeySource(ks),
		WithIssuer(testIssuer),
		WithAudiences(testAudience),
	}
	return NewVerifier(ks, append(base, opts...)...)
}

func newKeysFromKeyset(t *testing.T, keys ...testKey) *Keys {
	t.Helper()
	set := make(map[string]*rsa.PublicKey, len(keys))
	for _, k := range keys {
		set[k.kid] = &k.key.PublicKey
	}
	return NewKeysFromSet(set)
}

// ---------- Verifier ----------

func TestVerify_AcceptsPersonToken(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	id, err := v.Verify(context.Background(), signToken(t, k, nil))
	require.NoError(t, err)
	assert.Equal(t, "person-1", id.PersonID)
	assert.Equal(t, "tenant-1", id.TenantID)
	assert.Equal(t, "user-1", id.UserID)
	assert.Equal(t, testAudience, id.ClientID)
	assert.False(t, id.IsMachine)
}

func TestVerify_AcceptsMachineToken(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, k, func(c jwt.MapClaims) {
		c["sub"] = "ak_abcdefg"
		c["token_usage"] = "machine"
		c["user_id"] = "machine-user-1"
		delete(c, "client_id")
	})
	id, err := v.Verify(context.Background(), token)
	require.NoError(t, err)
	assert.True(t, id.IsMachine)
	assert.Equal(t, "machine-user-1", id.UserID)
	assert.Equal(t, "", id.PersonID, "机器令牌无自然人 ID")
}

func TestVerify_ParsesScopeAndActor(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, k, func(c jwt.MapClaims) {
		c["scope"] = "openid directory.read"
		c["act"] = map[string]any{"sub": "actor-9"}
	})
	id, err := v.Verify(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "directory.read"}, id.Scopes)
	assert.True(t, id.HasScope("directory.read"))
	assert.False(t, id.HasScope("directory.write"))
	assert.Equal(t, "actor-9", id.ActorID)
}

func TestVerify_ExpiredToken(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, k, func(c jwt.MapClaims) {
		c["exp"] = time.Now().Add(-time.Hour).Unix()
	})
	_, err := v.Verify(context.Background(), token)
	assert.ErrorIs(t, err, contract.ErrExpired)
}

// TestVerify_ClockSkewWithinLeeway 覆盖验收标准 3 的“时钟偏移 ±30s 内仍放行”。
func TestVerify_ClockSkewWithinLeeway(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	t.Run("expired 20s ago still accepted", func(t *testing.T) {
		token := signToken(t, k, func(c jwt.MapClaims) {
			c["exp"] = time.Now().Add(-20 * time.Second).Unix()
		})
		_, err := v.Verify(context.Background(), token)
		assert.NoError(t, err)
	})
	t.Run("nbf 20s in future still accepted", func(t *testing.T) {
		token := signToken(t, k, func(c jwt.MapClaims) {
			c["nbf"] = time.Now().Add(20 * time.Second).Unix()
		})
		_, err := v.Verify(context.Background(), token)
		assert.NoError(t, err)
	})
	t.Run("expired 5min ago rejected", func(t *testing.T) {
		token := signToken(t, k, func(c jwt.MapClaims) {
			c["exp"] = time.Now().Add(-5 * time.Minute).Unix()
		})
		_, err := v.Verify(context.Background(), token)
		assert.ErrorIs(t, err, contract.ErrExpired)
	})
}

func TestVerify_RejectsWrongIssuer(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, k, func(c jwt.MapClaims) { c["iss"] = "http://evil.example.com/oidc" })
	_, err := v.Verify(context.Background(), token)
	assert.ErrorIs(t, err, contract.ErrBadIssuer)
}

func TestVerify_RejectsWrongAudience(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, k, func(c jwt.MapClaims) { c["aud"] = "tenant_admin_web" })
	_, err := v.Verify(context.Background(), token)
	assert.ErrorIs(t, err, contract.ErrBadAudience)
}

// TestVerify_RejectsHS256Confusion 覆盖"HS256 混淆拒绝"这一基线判定行为。
func TestVerify_RejectsHS256Confusion(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	claims := jwt.MapClaims{
		"sub":       "person:person-1",
		"tenant_id": "tenant-1",
		"iss":       testIssuer,
		"aud":       testAudience,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "kid-1"
	signed, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	_, err = v.Verify(context.Background(), signed)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, nil))
}

func TestVerify_RejectsBadSignature(t *testing.T) {
	k := newTestKey(t, "kid-1")
	other := newTestKey(t, "kid-1") // 同 kid，不同密钥
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	_, err := v.Verify(context.Background(), signToken(t, other, nil))
	assert.ErrorIs(t, err, contract.ErrBadSignature)
}

func TestVerify_RejectsMissingKid(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	claims := jwt.MapClaims{
		"sub": "person:person-1", "tenant_id": "t", "iss": testIssuer,
		"aud": testAudience, "exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims) // 不设置 kid
	signed, err := token.SignedString(k.key)
	require.NoError(t, err)

	_, err = v.Verify(context.Background(), signed)
	assert.ErrorIs(t, err, contract.ErrMissingKid)
}

// TestVerify_RejectsForbiddenHeaders 覆盖本轮新增的收紧项：拒绝 jwk/jku/x5u。
func TestVerify_RejectsForbiddenHeaders(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	for _, header := range forbiddenHeaderKeys {
		t.Run(header, func(t *testing.T) {
			claims := jwt.MapClaims{
				"sub": "person:person-1", "tenant_id": "t", "iss": testIssuer,
				"aud": testAudience, "exp": time.Now().Add(time.Hour).Unix(),
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "kid-1"
			token.Header[header] = "https://evil.example.com/keys"
			signed, err := token.SignedString(k.key)
			require.NoError(t, err)

			_, err = v.Verify(context.Background(), signed)
			assert.ErrorIs(t, err, contract.ErrUnauthorizedHeader)
		})
	}
}

func TestVerify_RejectsUnknownKid(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, newTestKey(t, "kid-missing"), nil)
	_, err := v.Verify(context.Background(), token)
	assert.ErrorIs(t, err, contract.ErrUnknownKid)
}

// TestVerify_AllowsTokenWithoutTenant 锁定 tenant_id 的语义边界：
// 它是「租户作用域」上下文条件，不是令牌有效性条件——验签必须通过，
// 由需要租户的调用方（如目录 API）判 403，而不是在验签层冒成 401。
func TestVerify_AllowsTokenWithoutTenant(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, k, func(c jwt.MapClaims) { delete(c, "tenant_id") })
	identity, err := v.Verify(context.Background(), token)
	require.NoError(t, err)
	assert.Empty(t, identity.TenantID)
	assert.NotEmpty(t, identity.PersonID)
}

func TestVerify_RejectsTokenWithoutPersonOrMachine(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	token := signToken(t, k, func(c jwt.MapClaims) { c["sub"] = "client-abc" })
	_, err := v.Verify(context.Background(), token)
	assert.ErrorIs(t, err, contract.ErrMissingClaim)

	// 普通 OIDC 客户端的纯 client_credentials 令牌就是这一形态：sub/client_id 都是
	// client_id，既无 person: 前缀也无 token_usage=machine。它必须被拒——否则一个
	// 没有任何租户归属的令牌就能通过验签，任何"只验签、不判租户"的接口都会被它绕过。
	plainClientCredentials := signToken(t, k, func(c jwt.MapClaims) {
		c["sub"] = "svc-client"
		c["client_id"] = "svc-client"
		delete(c, "tenant_id")
		delete(c, "user_id")
	})
	_, err = v.Verify(context.Background(), plainClientCredentials)
	assert.ErrorIs(t, err, contract.ErrMissingClaim)
}

func TestVerify_RejectsMalformedAndEmpty(t *testing.T) {
	k := newTestKey(t, "kid-1")
	v := newTestVerifier(t, newKeysFromKeyset(t, k))

	_, err := v.Verify(context.Background(), "")
	assert.ErrorIs(t, err, contract.ErrMalformedToken)
	_, err = v.Verify(context.Background(), "not-a-jwt")
	assert.ErrorIs(t, err, contract.ErrMalformedToken)
}

func TestVerify_FailsClosedWithoutKeySource(t *testing.T) {
	v := NewVerifier(nil)
	_, err := v.Verify(context.Background(), "whatever")
	assert.ErrorIs(t, err, contract.ErrKeySourceUnavailable)
}

func TestIdentity_ContextRoundTrip(t *testing.T) {
	id := &Identity{UserID: "u1"}
	ctx := WithIdentity(context.Background(), id)
	assert.Equal(t, id, IdentityFrom(ctx))
	assert.Nil(t, IdentityFrom(context.Background()))
}

// ---------- KeySource ----------

func TestKeys_PrefetchAndCacheHit(t *testing.T) {
	k := newTestKey(t, "kid-1")
	var requests int32
	srv := jwksServer(t, func() []byte { return jwksBody(t, k) }, &requests)

	ks, err := NewKeys(context.Background(), srv.URL+"/jwks.json", WithMinRefreshInterval(time.Minute))
	require.NoError(t, err)
	defer ks.Close()
	require.Equal(t, int32(1), atomic.LoadInt32(&requests), "启动期必须预取一次")

	v := newTestVerifier(t, ks)
	_, err = v.Verify(context.Background(), signToken(t, k, nil))
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requests), "命中缓存时请求路径零网络调用")
}

// TestKeys_UnknownKidTriggersRefresh 覆盖"未知 kid 立即刷新一次"。
func TestKeys_UnknownKidTriggersRefresh(t *testing.T) {
	oldKey := newTestKey(t, "kid-old")
	newKey := newTestKey(t, "kid-new")
	var requests int32
	current := jwksBody(t, oldKey)
	srv := jwksServer(t, func() []byte { return current }, &requests)

	ks, err := NewKeys(context.Background(), srv.URL+"/jwks.json", WithMinRefreshInterval(0))
	require.NoError(t, err)
	defer ks.Close()

	// 模拟 OP 例行轮换：JWKS 追加新 key（旧 key 仍在发布）。
	current = jwksBody(t, oldKey, newKey)

	id, err := newTestVerifier(t, ks).Verify(context.Background(), signToken(t, newKey, nil))
	require.NoError(t, err, "未知 kid 应立即刷新后放行")
	assert.Equal(t, "user-1", id.UserID)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&requests), int32(2))
}

// TestKeys_RefreshRateLimited 覆盖"未知 kid 刷新限速 1 次/分钟"。
func TestKeys_RefreshRateLimited(t *testing.T) {
	oldKey := newTestKey(t, "kid-old")
	var requests int32
	srv := jwksServer(t, func() []byte { return jwksBody(t, oldKey) }, &requests)

	now := time.Now()
	ks, err := NewKeys(context.Background(), srv.URL+"/jwks.json",
		WithMinRefreshInterval(time.Minute),
		withNow(func() time.Time { return now }),
	)
	require.NoError(t, err)
	defer ks.Close()
	require.Equal(t, int32(1), atomic.LoadInt32(&requests))

	// 未知 kid：窗口内不得再发请求
	for i := 0; i < 5; i++ {
		_, err := ks.PublicKey(context.Background(), "kid-unknown")
		assert.ErrorIs(t, err, contract.ErrUnknownKid)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&requests), "限速窗口内不得重复拉取")

	// 窗口过后允许一次
	now = now.Add(2 * time.Minute)
	_, err = ks.PublicKey(context.Background(), "kid-unknown")
	assert.ErrorIs(t, err, contract.ErrUnknownKid)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requests))
}

// TestKeys_RefreshFailureKeepsOldKeys 覆盖 stale-while-error。
func TestKeys_RefreshFailureKeepsOldKeys(t *testing.T) {
	k := newTestKey(t, "kid-1")
	var failing int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.LoadInt32(&failing) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksBody(t, k))
	}))
	defer srv.Close()

	ks, err := NewKeys(context.Background(), srv.URL+"/jwks.json", WithMinRefreshInterval(0))
	require.NoError(t, err)
	defer ks.Close()

	atomic.StoreInt32(&failing, 1)
	// 未知 kid 触发刷新失败，但已知 kid 仍可用（保留旧 key）。
	_, err = ks.PublicKey(context.Background(), "kid-unknown")
	assert.ErrorIs(t, err, contract.ErrUnknownKid)
	key, err := ks.PublicKey(context.Background(), "kid-1")
	require.NoError(t, err, "拉取失败必须保留旧 key")
	assert.NotNil(t, key)
}

// TestKeys_MaxKeyAgeFailClosed 覆盖"陈旧密钥超过上界即不可用"。
func TestKeys_MaxKeyAgeFailClosed(t *testing.T) {
	k := newTestKey(t, "kid-1")
	var failing int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.LoadInt32(&failing) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksBody(t, k))
	}))
	defer srv.Close()

	now := time.Now()
	ks, err := NewKeys(context.Background(), srv.URL+"/jwks.json",
		WithMinRefreshInterval(0),
		WithMaxKeyAge(time.Minute),
		withNow(func() time.Time { return now }),
	)
	require.NoError(t, err)
	defer ks.Close()

	atomic.StoreInt32(&failing, 1)
	now = now.Add(2 * time.Minute) // 超过 maxKeyAge
	// 缓存已陈旧且无法刷新：任何 kid（含未知）都按"密钥源不可用"fail-closed，
	// 避免用无限陈旧的公钥继续放行。
	_, err = ks.PublicKey(context.Background(), "kid-unknown")
	assert.ErrorIs(t, err, contract.ErrKeySourceUnavailable)

	// 已知 kid 同样不可用。
	_, err = ks.PublicKey(context.Background(), "kid-1")
	assert.ErrorIs(t, err, contract.ErrKeySourceUnavailable)
}

func TestKeys_FallbackFileWhenRemoteDown(t *testing.T) {
	k := newTestKey(t, "kid-1")
	dir := t.TempDir()
	fallback := dir + "/jwks.json"
	require.NoError(t, writeFile(fallback, jwksBody(t, k)))

	// 远端不可用（端口上无服务）。
	ks, err := NewKeys(context.Background(), "http://127.0.0.1:1/jwks.json", WithJWKSFallback(fallback))
	require.NoError(t, err, "有可用落盘兜底时启动不应失败")
	defer ks.Close()

	v := newTestVerifier(t, ks)
	_, err = v.Verify(context.Background(), signToken(t, k, nil))
	require.NoError(t, err)
}

func TestKeys_PrefetchFailureFailsFast(t *testing.T) {
	_, err := NewKeys(context.Background(), "http://127.0.0.1:1/jwks.json")
	require.Error(t, err, "既无远端也无兜底必须 fail-fast")
}

func TestKeys_PersistFallback(t *testing.T) {
	k := newTestKey(t, "kid-1")
	srv := jwksServer(t, func() []byte { return jwksBody(t, k) }, nil)
	fallback := t.TempDir() + "/jwks.json"

	ks, err := NewKeys(context.Background(), srv.URL+"/jwks.json", WithJWKSFallback(fallback))
	require.NoError(t, err)
	defer ks.Close()

	raw, err := readFile(fallback)
	require.NoError(t, err, "成功预取后必须写落盘兜底")
	assert.Contains(t, string(raw), "kid-1")
}

func TestParseJWKS_SkipsNonRSAAndEncryptionKeys(t *testing.T) {
	body := []byte(`{"keys":[
		{"kty":"EC","kid":"ec-1","crv":"P-256","x":"a","y":"b"},
		{"kty":"RSA","kid":"enc-1","use":"enc","n":"AQAB","e":"AQAB"},
		{"kty":"RSA","kid":"","n":"AQAB","e":"AQAB"}
	]}`)
	keys, err := parseJWKS(body)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestParseJWKS_InvalidDocument(t *testing.T) {
	_, err := parseJWKS([]byte("not json"))
	require.Error(t, err)
	_, err = parseJWKS([]byte(`{"keys":[]}`))
	require.NoError(t, err)
}

func TestRsaPublicKeyFromJWK_RejectsBadExponent(t *testing.T) {
	_, err := rsaPublicKeyFromJWK(jwk{Kty: "RSA", Kid: "k", N: "AQAB", E: ""})
	require.Error(t, err)
	_, err = rsaPublicKeyFromJWK(jwk{Kty: "RSA", Kid: "k", N: "AQAB", E: "AA"})
	require.Error(t, err, "指数为 0 必须拒绝")
}

func TestResolveJWKSURLCandidates_NoNetworkForExplicitEndpoint(t *testing.T) {
	cfg := newKeysConfig(nil)
	// 显式端点原样使用：{issuer}/keys 与 .../jwks.json 都不得再做推导/网络请求。
	assert.Equal(t, []string{"http://h/oidc/keys"}, jwksURLCandidates(context.Background(), "http://h/oidc/keys", cfg))
	assert.Equal(t, []string{"http://h/custom/jwks.json"}, jwksURLCandidates(context.Background(), "http://h/custom/jwks.json", cfg))
	assert.Nil(t, jwksURLCandidates(context.Background(), "   ", cfg))
}

// TestResolveJWKSURLCandidates_IssuerPrefersDiscovery 断言 issuer 形态的候选顺序：
// discovery 的 jwks_uri 优先，其后是 {issuer}/keys 与 {issuer}/.well-known/jwks.json。
func TestResolveJWKSURLCandidates_IssuerPrefersDiscovery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oidc"+discoveryPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"jwks_uri":"http://example.com/jwks.json"}`))
	}))
	defer srv.Close()

	got := jwksURLCandidates(context.Background(), srv.URL+"/oidc", newKeysConfig(nil))
	assert.Equal(t,
		[]string{"http://example.com/jwks.json", srv.URL + "/oidc/keys", srv.URL + "/oidc/.well-known/jwks.json"},
		got)
}

// TestResolveJWKSURLCandidates_IssuerWithoutDiscovery 断言 discovery 不可用时
// 仍保留 {issuer}/keys（本仓 OP 的发布路径）与历史路径两个回退候选。
func TestResolveJWKSURLCandidates_IssuerWithoutDiscovery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	got := jwksURLCandidates(context.Background(), srv.URL+"/oidc/", newKeysConfig(nil))
	assert.Equal(t, []string{srv.URL + "/oidc/keys", srv.URL + "/oidc/.well-known/jwks.json"}, got)
}

// TestKeys_IssuerResolvesKeysEndpoint 回归本仓 OP 的真实形态：issuer 下的 JWKS
// 发布在 {issuer}/keys（zitadel/oidc 默认端点，discovery 的 jwks_uri 指向它）。
// 旧实现会去取 {issuer}/.well-known/jwks.json 并拿到 404，导致启动预取 fail-fast。
func TestKeys_IssuerResolvesKeysEndpoint(t *testing.T) {
	k := newTestKey(t, "kid-1")
	var keysHits, wellKnownHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oidc" + discoveryPath:
			_, _ = w.Write([]byte(`{"jwks_uri":"http://` + r.Host + `/oidc/keys"}`))
		case "/oidc/keys":
			atomic.AddInt32(&keysHits, 1)
			_, _ = w.Write(jwksBody(t, k))
		default:
			if r.URL.Path == "/oidc"+wellKnownJWKSPath {
				atomic.AddInt32(&wellKnownHits, 1)
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ks, err := NewKeys(context.Background(), srv.URL+"/oidc")
	require.NoError(t, err)
	defer ks.Close()
	assert.Equal(t, int32(1), atomic.LoadInt32(&keysHits), "预取必须命中 discovery 声明的 jwks_uri")
	assert.Equal(t, int32(0), atomic.LoadInt32(&wellKnownHits), "命中后不得再猜其他端点")

	_, err = newTestVerifier(t, ks).Verify(context.Background(), signToken(t, k, nil))
	require.NoError(t, err)
}

// TestKeys_IssuerFallsBackToKeysEndpoint 覆盖 discovery 不可用时的回退：
// 只有 {issuer}/keys 可用时也必须启动成功。
func TestKeys_IssuerFallsBackToKeysEndpoint(t *testing.T) {
	k := newTestKey(t, "kid-1")
	var keysHits, wellKnownHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/keys" {
			atomic.AddInt32(&keysHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(jwksBody(t, k))
			return
		}
		if r.URL.Path == "/oidc"+wellKnownJWKSPath {
			atomic.AddInt32(&wellKnownHits, 1)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ks, err := NewKeys(context.Background(), srv.URL+"/oidc")
	require.NoError(t, err)
	defer ks.Close()
	assert.Equal(t, int32(1), atomic.LoadInt32(&keysHits))
	assert.Equal(t, int32(0), atomic.LoadInt32(&wellKnownHits), "命中 /keys 后不得继续猜路径")

	key, err := ks.PublicKey(context.Background(), "kid-1")
	require.NoError(t, err)
	require.NotNil(t, key)
}

// TestKeys_KeepsResolvedEndpoint 断言端点一经确定即固化：后续刷新只打同一个地址，
// 不再重复走 discovery 与回退候选。
func TestKeys_KeepsResolvedEndpoint(t *testing.T) {
	k := newTestKey(t, "kid-1")
	var discoveryHits, keysHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oidc" + discoveryPath:
			atomic.AddInt32(&discoveryHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jwks_uri":"http://` + r.Host + `/oidc/keys"}`))
		case "/oidc/keys":
			atomic.AddInt32(&keysHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(jwksBody(t, k))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ks, err := NewKeys(context.Background(), srv.URL+"/oidc", WithMinRefreshInterval(0))
	require.NoError(t, err)
	defer ks.Close()

	// 未知 kid 触发一次即时刷新：应直接重试已固化的 /oidc/keys。
	_, err = ks.PublicKey(context.Background(), "kid-unknown")
	assert.ErrorIs(t, err, contract.ErrUnknownKid)
	assert.Equal(t, int32(1), atomic.LoadInt32(&discoveryHits), "discovery 只做一次")
	assert.Equal(t, int32(2), atomic.LoadInt32(&keysHits), "刷新复用已固化端点")
}

func TestDiscoveryJWKSURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != discoveryPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"jwks_uri":"http://example.com/jwks.json"}`))
	}))
	defer srv.Close()

	got, err := DiscoveryJWKSURL(context.Background(), srv.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/jwks.json", got)

	_, err = DiscoveryJWKSURL(context.Background(), "", nil)
	require.Error(t, err)
}

func TestNewKeysFromSet_UnknownKid(t *testing.T) {
	k := newTestKey(t, "kid-1")
	ks := newKeysFromKeyset(t, k)
	defer ks.Close()

	key, err := ks.PublicKey(context.Background(), "kid-1")
	require.NoError(t, err)
	require.NotNil(t, key)

	_, err = ks.PublicKey(context.Background(), "nope")
	assert.ErrorIs(t, err, contract.ErrUnknownKid)
	_, err = ks.PublicKey(context.Background(), "")
	assert.ErrorIs(t, err, contract.ErrMissingKid)
}

// BenchmarkVerify_Keyset 量化"SDK 验签相对无鉴权的增量"（验收标准 5 的口径）：
// 内存 kid 查找 + 2048-bit RSA 本地验签，无网络调用。
func BenchmarkVerify_Keyset(b *testing.B) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}
	ks := NewKeysFromSet(map[string]*rsa.PublicKey{"kid-1": &key.PublicKey})
	verifier := NewVerifier(ks, WithIssuer(testIssuer), WithAudiences(testAudience))

	claims := jwt.MapClaims{
		"sub": "person:p1", "tenant_id": "t1", "user_id": "u1",
		"iss": testIssuer, "aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "kid-1"
	raw, err := token.SignedString(key)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := verifier.Verify(ctx, raw); err != nil {
			b.Fatal(err)
		}
	}
}

// writeFile / readFile 是小工具，避免测试文件里反复出现 os 调用样板。
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
