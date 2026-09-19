package goidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer  = "http://localhost:8081/oidc"
	testClient  = "platform_admin_web"
	testSID     = "sess-12345"
	testSubject = "person:42"
)

// testClaims 是与 LogoutTokenClaims 结构一致的本地构造体（不引入 zitadel 依赖）。
type testClaims struct {
	jwt.RegisteredClaims
	SessionID string         `json:"sid,omitempty"`
	Events    map[string]any `json:"events,omitempty"`
}

// testKeySource 返回"单 key"KeySource（kid 与 token 头一致），
// 与本包生产接线（rp.NewKeysFromSet / JWKS）保持同一取键口径。
func testKeySource(m map[string]*rsa.PublicKey) KeySource { return keyMapSource(m) }

type keyMapSource map[string]*rsa.PublicKey

func (m keyMapSource) PublicKey(_ context.Context, kid string) (*rsa.PublicKey, error) {
	key, ok := m[kid]
	if !ok {
		return nil, errors.New("unknown kid: " + kid)
	}
	return key, nil
}

// newTestKeyAndSource 生成一把测试私钥，并返回可按 kid 取到其公钥的 KeySource。
func newTestKeyAndSource(t *testing.T) (*rsa.PrivateKey, KeySource, string) {
	t.Helper()
	privKey := newTestKey(t)
	kid := "test-kid"
	return privKey, testKeySource(map[string]*rsa.PublicKey{kid: &privKey.PublicKey}), kid
}

func buildTestLogoutTokenWithKID(t *testing.T, privKey *rsa.PrivateKey, kid, issuer, clientID, sid, sub string, withEvent bool) string {
	t.Helper()
	return signLogoutToken(t, privKey, kid, issuer, clientID, sid, sub, withEvent)
}

func signLogoutToken(t *testing.T, privKey *rsa.PrivateKey, kid, issuer, clientID, sid, sub string, withEvent bool) string {
	t.Helper()
	events := map[string]any{}
	if withEvent {
		events[BackChannelLogoutEventURI] = map[string]any{}
	}
	claims := testClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   sub,
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        "jti-" + strings.Repeat("1", 12),
		},
		SessionID: sid,
		Events:    events,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenStr, err := token.SignedString(privKey)
	require.NoError(t, err)
	return tokenStr
}

func buildTestLogoutToken(t *testing.T, privKey *rsa.PrivateKey, issuer, clientID, sid, sub string, withEvent bool) string {
	t.Helper()
	return signLogoutToken(t, privKey, "", issuer, clientID, sid, sub, withEvent)
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return privKey
}

func TestParseLogoutToken_Valid(t *testing.T) {
	privKey, source, kid := newTestKeyAndSource(t)
	tokenStr := buildTestLogoutTokenWithKID(t, privKey, kid, testIssuer, testClient, testSID, testSubject, true)

	claims, err := ParseLogoutToken(tokenStr, source, testIssuer, testClient)
	require.NoError(t, err)
	assert.Equal(t, testSubject, claims.Subject)
	assert.Equal(t, testSID, claims.SessionID)
	assert.True(t, claims.HasBackChannelLogoutEvent())
	assert.NotEmpty(t, claims.ID)
}

func TestParseLogoutToken_InvalidSignature(t *testing.T) {
	privKey, _, kid := newTestKeyAndSource(t)
	otherPrivKey := newTestKey(t)
	tokenStr := buildTestLogoutTokenWithKID(t, privKey, kid, testIssuer, testClient, testSID, testSubject, true)

	_, err := ParseLogoutToken(tokenStr, testKeySource(map[string]*rsa.PublicKey{kid: &otherPrivKey.PublicKey}), testIssuer, testClient)
	require.Error(t, err)
}

func TestParseLogoutToken_WrongIssuer(t *testing.T) {
	privKey, source, kid := newTestKeyAndSource(t)
	tokenStr := buildTestLogoutTokenWithKID(t, privKey, kid, "http://evil.example/oidc", testClient, testSID, testSubject, true)

	_, err := ParseLogoutToken(tokenStr, source, testIssuer, testClient)
	require.Error(t, err)
}

func TestParseLogoutToken_WrongAudience(t *testing.T) {
	privKey, source, kid := newTestKeyAndSource(t)
	tokenStr := buildTestLogoutTokenWithKID(t, privKey, kid, testIssuer, "some-other-rp", testSID, testSubject, true)

	_, err := ParseLogoutToken(tokenStr, source, testIssuer, testClient)
	require.Error(t, err)
}

func TestParseLogoutToken_MissingEvent(t *testing.T) {
	privKey, source, kid := newTestKeyAndSource(t)
	tokenStr := buildTestLogoutTokenWithKID(t, privKey, kid, testIssuer, testClient, testSID, testSubject, false)

	_, err := ParseLogoutToken(tokenStr, source, testIssuer, testClient)
	require.Error(t, err)
}

func TestBackChannelLogoutHandler_ValidToken_200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privKey, source, kid := newTestKeyAndSource(t)

	var revoked []*LogoutTokenClaims
	h := NewBackChannelLogoutHandler(source, testIssuer, testClient,
		func(ctx context.Context, claims *LogoutTokenClaims) error {
			revoked = append(revoked, claims)
			return nil
		})

	engine := gin.New()
	engine.POST("/bc-logout", h.Handler())

	tokenStr := buildTestLogoutTokenWithKID(t, privKey, kid, testIssuer, testClient, testSID, testSubject, true)
	form := url.Values{}
	form.Set("logout_token", tokenStr)
	req := httptest.NewRequest(http.MethodPost, "/bc-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, revoked, 1)
	assert.Equal(t, testSID, revoked[0].SessionID)

	recent := h.Recent()
	require.Len(t, recent, 1)
	assert.True(t, recent[0].Valid)
	assert.Equal(t, testSID, recent[0].SID)
}

func TestBackChannelLogoutHandler_InvalidToken_400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, source, _ := newTestKeyAndSource(t)

	h := NewBackChannelLogoutHandler(source, testIssuer, testClient, nil)
	engine := gin.New()
	engine.POST("/bc-logout", h.Handler())

	form := url.Values{}
	form.Set("logout_token", "not-a-jwt")
	req := httptest.NewRequest(http.MethodPost, "/bc-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.False(t, h.Recent()[0].Valid)
}

func TestBackChannelLogoutHandler_RevokerError_500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privKey, source, kid := newTestKeyAndSource(t)

	h := NewBackChannelLogoutHandler(source, testIssuer, testClient,
		func(ctx context.Context, claims *LogoutTokenClaims) error {
			return errors.New("revoker boom")
		})
	engine := gin.New()
	engine.POST("/bc-logout", h.Handler())

	tokenStr := buildTestLogoutTokenWithKID(t, privKey, kid, testIssuer, testClient, testSID, testSubject, true)
	form := url.Values{}
	form.Set("logout_token", tokenStr)
	req := httptest.NewRequest(http.MethodPost, "/bc-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}
