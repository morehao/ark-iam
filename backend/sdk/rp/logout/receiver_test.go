package logout

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	testClientID = "platform_admin_web"
	testEventURI = contract.BackChannelLogoutEventURI
)

type staticJWKS struct {
	kid string
	key *rsa.PublicKey
	err error
}

func (s staticJWKS) PublicKey(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if s.err != nil {
		return nil, s.err
	}
	if kid != s.kid {
		return nil, contract.ErrUnknownKid
	}
	return s.key, nil
}

func newReceiverTestKey(t *testing.T) (*rsa.PrivateKey, staticJWKS) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key, staticJWKS{kid: "kid-1", key: &key.PublicKey}
}

// buildLogoutToken 构造 logout_token；mutate 可覆盖/删除声明用于负例。
func buildLogoutToken(t *testing.T, key *rsa.PrivateKey, mutate func(jwt.MapClaims)) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":    testIssuer,
		"aud":    testClientID,
		"sub":    "person:person-1",
		"sid":    "session-1",
		"jti":    "jti-1",
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
		"events": map[string]any{testEventURI: map[string]any{}},
	}
	if mutate != nil {
		mutate(claims)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "kid-1"
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func postLogout(t *testing.T, h http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	form.Set("logout_token", token)
	req := httptest.NewRequest(http.MethodPost, "/oidc/bc-logout/app", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestReceiver_AcceptsValidToken(t *testing.T) {
	key, ks := newReceiverTestKey(t)
	var calls int32
	rec := NewReceiver(
		WithIssuer(testIssuer),
		WithClientID(testClientID),
		WithKeySource(ks),
		WithSessionRevoker(func(_ context.Context, claims *contract.LogoutTokenClaims) error {
			atomic.AddInt32(&calls, 1)
			assert.Equal(t, "person:person-1", claims.Subject)
			assert.Equal(t, "session-1", claims.SessionID)
			return nil
		}),
	)
	w := postLogout(t, rec, buildLogoutToken(t, key, nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestReceiver_RejectsInvalidTokens(t *testing.T) {
	key, ks := newReceiverTestKey(t)
	otherKey, _ := newReceiverTestKey(t)

	cases := []struct {
		name   string
		mutate func(jwt.MapClaims)
		key    *rsa.PrivateKey
	}{
		{name: "wrong issuer", mutate: func(c jwt.MapClaims) { c["iss"] = "http://evil/oidc" }},
		{name: "wrong audience", mutate: func(c jwt.MapClaims) { c["aud"] = "other_client" }},
		{name: "expired", mutate: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }},
		{name: "missing exp", mutate: func(c jwt.MapClaims) { delete(c, "exp") }},
		{name: "missing iat", mutate: func(c jwt.MapClaims) { delete(c, "iat") }},
		{name: "missing jti", mutate: func(c jwt.MapClaims) { delete(c, "jti") }},
		{name: "missing sub", mutate: func(c jwt.MapClaims) { delete(c, "sub") }},
		{name: "missing events", mutate: func(c jwt.MapClaims) { delete(c, "events") }},
		{name: "wrong event", mutate: func(c jwt.MapClaims) { c["events"] = map[string]any{"other": 1} }},
		{name: "nonce present", mutate: func(c jwt.MapClaims) { c["nonce"] = "n-1" }},
		{name: "bad signature", key: otherKey},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			signingKey := key
			if tc.key != nil {
				signingKey = tc.key
			}
			var revoked int32
			rec := NewReceiver(
				WithIssuer(testIssuer),
				WithClientID(testClientID),
				WithKeySource(ks),
				WithSessionRevoker(func(context.Context, *contract.LogoutTokenClaims) error {
					atomic.AddInt32(&revoked, 1)
					return nil
				}),
			)
			w := postLogout(t, rec, buildLogoutToken(t, signingKey, tc.mutate))
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, int32(0), atomic.LoadInt32(&revoked), "无效 token 不得触发本地登出")
		})
	}
}

func TestReceiver_RejectsMissingKidAndForbiddenHeader(t *testing.T) {
	key, ks := newReceiverTestKey(t)
	rec := NewReceiver(WithIssuer(testIssuer), WithClientID(testClientID), WithKeySource(ks))

	for _, header := range []string{"jwk", "jku", "x5u"} {
		t.Run("forbidden "+header, func(t *testing.T) {
			claims := jwt.MapClaims{
				"iss": testIssuer, "aud": testClientID, "sub": "person:1", "jti": "j1",
				"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
				"events": map[string]any{testEventURI: map[string]any{}},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "kid-1"
			token.Header[header] = "https://evil/keys"
			signed, err := token.SignedString(key)
			require.NoError(t, err)
			w := postLogout(t, rec, signed)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}

	t.Run("missing kid", func(t *testing.T) {
		claims := jwt.MapClaims{
			"iss": testIssuer, "aud": testClientID, "sub": "person:1", "jti": "j1",
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			"events": map[string]any{testEventURI: map[string]any{}},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signed, err := token.SignedString(key)
		require.NoError(t, err)
		w := postLogout(t, rec, signed)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestReceiver_RevokerFailureAllowsRetry 覆盖验收标准 11：撤销失败必须 500，
// 且**不记 jti**，使 OP 的重投能真正撤销。
func TestReceiver_RevokerFailureAllowsRetry(t *testing.T) {
	key, ks := newReceiverTestKey(t)
	var attempts int32
	rec := NewReceiver(
		WithIssuer(testIssuer),
		WithClientID(testClientID),
		WithKeySource(ks),
		WithSessionRevoker(func(context.Context, *contract.LogoutTokenClaims) error {
			if atomic.AddInt32(&attempts, 1) == 1 {
				return errors.New("db down")
			}
			return nil
		}),
	)
	token := buildLogoutToken(t, key, nil)

	require.Equal(t, http.StatusInternalServerError, postLogout(t, rec, token).Code)
	require.Equal(t, http.StatusOK, postLogout(t, rec, token).Code, "重投必须真正执行撤销")
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

// TestReceiver_DuplicateDeliveryIsIdempotent 覆盖"重复投递不产生二次副作用"。
func TestReceiver_DuplicateDeliveryIsIdempotent(t *testing.T) {
	key, ks := newReceiverTestKey(t)
	var calls int32
	rec := NewReceiver(
		WithIssuer(testIssuer),
		WithClientID(testClientID),
		WithKeySource(ks),
		WithSessionRevoker(func(context.Context, *contract.LogoutTokenClaims) error {
			atomic.AddInt32(&calls, 1)
			return nil
		}),
	)
	token := buildLogoutToken(t, key, nil)

	require.Equal(t, http.StatusOK, postLogout(t, rec, token).Code)
	require.Equal(t, http.StatusOK, postLogout(t, rec, token).Code, "重复投递仍返回 200")
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "重复投递不得二次执行本地登出")

	// 不同 jti 是新的一次登出。
	other := buildLogoutToken(t, key, func(c jwt.MapClaims) { c["jti"] = "jti-2" })
	require.Equal(t, http.StatusOK, postLogout(t, rec, other).Code)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestReceiver_RejectsNonPost(t *testing.T) {
	_, ks := newReceiverTestKey(t)
	rec := NewReceiver(WithIssuer(testIssuer), WithClientID(testClientID), WithKeySource(ks))
	req := httptest.NewRequest(http.MethodGet, "/oidc/bc-logout/app", nil)
	w := httptest.NewRecorder()
	rec.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestReceiver_FailsClosedWithoutConfig(t *testing.T) {
	rec := NewReceiver()
	w := postLogout(t, rec, "whatever")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMemoryJTIStore_Expiry(t *testing.T) {
	now := time.Now()
	store := NewMemoryJTIStore()
	store.now = func() time.Time { return now }
	ctx := context.Background()

	assert.False(t, store.Seen(ctx, "j1"))
	require.NoError(t, store.Record(ctx, "j1", now.Add(time.Minute)))
	assert.True(t, store.Seen(ctx, "j1"))

	now = now.Add(2 * time.Minute)
	assert.False(t, store.Seen(ctx, "j1"), "过期记录必须失效")
}

func TestHashJTI(t *testing.T) {
	assert.Equal(t, HashJTI("j1"), HashJTI("j1"))
	assert.NotEqual(t, HashJTI("j1"), HashJTI("j2"))
	assert.Len(t, HashJTI("j1"), 64)
}
