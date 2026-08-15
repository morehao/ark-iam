package svcoidc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	appconfig "github.com/morehao/ark-iam/auth/config"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newProtocolConformanceStore 构造带 SQLite person/user/refresh/client DAO 的 persistent store。
func newProtocolConformanceStore(t *testing.T, migrate ...any) (*PersistentStore, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:conformance_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	entities := append([]any{&model.PersonEntity{}, &model.UserEntity{}, &model.RefreshTokenEntity{}, &model.ApplicationClientEntity{}}, migrate...)
	if err := db.AutoMigrate(entities...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	ps := NewPersistentStore()
	ps.personDao = func(opts ...dao.DaoOption) *dao.PersonDao {
		return dao.NewPersonDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}
	ps.userDao = func(opts ...dao.DaoOption) *dao.UserDao {
		return dao.NewUserDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}
	ps.refreshTokenDao = func(opts ...dao.DaoOption) *dao.RefreshTokenDao {
		return dao.NewRefreshTokenDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}
	ps.applicationClientDao = func(opts ...dao.DaoOption) *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}
	ps.applicationClientSecretDao = func(opts ...dao.DaoOption) *dao.ApplicationClientSecretDao {
		return dao.NewApplicationClientSecretDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}
	return ps, db
}

func seedPerson(t *testing.T, db *gorm.DB, id string, email string) {
	t.Helper()
	emailPtr := model.StrPtr(email)
	if err := db.Create(&model.PersonEntity{
		BaseEntity:        gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		Name:              "test-user",
		Username:          model.StrPtr("user-" + fmt.Sprint(id)),
		PrimaryEmail:      emailPtr,
		Profile:           json.RawMessage(`{}`),
		CustomData:        json.RawMessage(`{}`),
		PasswordEncrypted: "hash",
	}).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
}

func seedUser(t *testing.T, db *gorm.DB, id, personID, tenantID string) {
	t.Helper()
	now := time.Now()
	if err := db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		PersonID:   personID,
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
		JoinedAt:   &now,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// TestRefreshTokenPersistsAndRestoresScopeAMRAuthTime 覆盖 H2：
// refresh token 必须持久化授权 scope/amr/auth_time，刷新时原样还原（RFC 6749 §6）。
func TestRefreshTokenPersistsAndRestoresScopeAMRAuthTime(t *testing.T) {
	ps, db := newProtocolConformanceStore(t)
	seedPerson(t, db, "88", "person@example.com")
	seedUser(t, db, "21", "88", "7")

	authTime := time.Unix(1710000000, 0)
	authReq := &AuthRequest{
		Subject:   buildOIDCSubject("88"),
		ClientID:  "client-1",
		TenantID:  "7",
		SessionID: "sid-1",
		Scopes:    []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail, "offline_access"},
		AMR:       []string{"pwd"},
		AuthTime:  authTime,
	}

	ctx := context.Background()
	_, refreshToken, _, err := ps.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}

	var stored model.RefreshTokenEntity
	if err := db.Where("token = ?", hashToken(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	gotScopes := decodeJSONStringSlice(stored.Scopes)
	if len(gotScopes) != 4 || gotScopes[3] != "offline_access" {
		t.Fatalf("expected persisted scopes [openid profile email offline_access], got %v", gotScopes)
	}
	gotAMR := decodeJSONStringSlice(stored.AMR)
	if len(gotAMR) != 1 || gotAMR[0] != "pwd" {
		t.Fatalf("expected persisted amr [pwd], got %v", gotAMR)
	}
	if stored.AuthTime == nil || !stored.AuthTime.Equal(authTime) {
		t.Fatalf("expected persisted auth_time %v, got %v", authTime, stored.AuthTime)
	}

	restored, err := ps.TokenRequestByRefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("TokenRequestByRefreshToken failed: %v", err)
	}
	if got := restored.GetScopes(); len(got) != 4 || got[3] != "offline_access" {
		t.Fatalf("expected restored scopes to include offline_access, got %v", got)
	}
	if got := restored.GetAMR(); len(got) != 1 || got[0] != "pwd" {
		t.Fatalf("expected restored amr [pwd], got %v", got)
	}
	if !restored.GetAuthTime().Equal(authTime) {
		t.Fatalf("expected restored auth_time %v, got %v", authTime, restored.GetAuthTime())
	}
	rr, ok := restored.(*refreshTokenRequest)
	if !ok || rr.GetSessionID() != "sid-1" {
		t.Fatalf("expected restored session id sid-1, got %+v", rr)
	}
}

// TestRefreshTokenTTLUsesClientConfig 覆盖 H2：refresh token 有效期按 client 配置的 refresh_token_ttl。
func TestRefreshTokenTTLUsesClientConfig(t *testing.T) {
	ps, db := newProtocolConformanceStore(t)
	seedPerson(t, db, "88", "person@example.com")
	seedUser(t, db, "21", "88", "7")
	if err := db.Create(&model.ApplicationClientEntity{
		BaseEntity:              gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}},
		Code:                   "client-ttl",
		RedirectURIs:            datatypes.JSON("[]"),
		PostLogoutRedirectURIs:  datatypes.JSON("[]"),
		GrantTypes:              datatypes.JSON(`["authorization_code"]`),
		ResponseTypes:           datatypes.JSON(`["code"]`),
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodBasic,
		AllowedOrigins:          datatypes.JSON("[]"),
		DefaultScopes:           datatypes.JSON(`["openid"]`),
		AccessTokenTTL:          900,
		RefreshTokenTTL:         7200,
		Status:                  model.ApplicationClientStatusEnable,
		Type:                    model.ApplicationClientTypeFirstParty,
	}).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}

	authReq := &AuthRequest{
		Subject:  buildOIDCSubject("88"),
		ClientID: "client-ttl",
		TenantID: "7",
		Scopes:   []string{oidc.ScopeOpenID},
		AMR:      []string{"pwd"},
	}
	ctx := context.Background()
	_, refreshToken, _, err := ps.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}
	var stored model.RefreshTokenEntity
	if err := db.Where("token = ?", hashToken(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	ttlSeconds := int64(stored.ExpiredAt.Sub(time.Now()).Seconds())
	if ttlSeconds < 7100 || ttlSeconds > 7300 {
		t.Fatalf("expected refresh token ttl ~7200s (client config), got %d", ttlSeconds)
	}
}

// TestSetIntrospectionFromTokenReturnsFullResponse 覆盖 M1：
// introspection 必须返回 scope/client_id/sub/exp/iat/token_type/username 及私有声明。
func TestSetIntrospectionFromTokenReturnsFullResponse(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ps, db := newProtocolConformanceStore(t)
	seedPerson(t, db, "88", "person@example.com")

	tokenID := "at-introspect-test"
	issuedAt := time.Now().Add(-time.Minute)
	expiresAt := time.Now().Add(15 * time.Minute)
	storeAccessTokenMeta(context.Background(), tokenID, accessTokenMeta{
		Subject:    buildOIDCSubject("88"),
		ClientID:   "client-1",
		Scopes:     []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		IssuedAt:   issuedAt,
		ExpiresAt:  expiresAt,
		TenantID:   "7",
		SessionID:  "sid-1",
		TokenUsage: "",
	})

	resp := &oidc.IntrospectionResponse{}
	if err := ps.SetIntrospectionFromToken(context.Background(), resp, tokenID, buildOIDCSubject("88"), "client-1"); err != nil {
		t.Fatalf("SetIntrospectionFromToken failed: %v", err)
	}
	if len(resp.Scope) != 2 {
		t.Fatalf("expected 2 scopes in introspection, got %v", resp.Scope)
	}
	if resp.ClientID != "client-1" {
		t.Fatalf("expected client_id client-1, got %q", resp.ClientID)
	}
	if resp.TokenType != oidc.BearerToken {
		t.Fatalf("expected token_type Bearer, got %q", resp.TokenType)
	}
	if resp.Subject != buildOIDCSubject("88") {
		t.Fatalf("expected subject person:88, got %q", resp.Subject)
	}
	if !resp.Expiration.AsTime().Equal(expiresAt.Truncate(time.Second)) {
		t.Fatalf("expected exp %v, got %v", expiresAt.Truncate(time.Second), resp.Expiration.AsTime())
	}
	if resp.Username != "user-88" {
		t.Fatalf("expected username user-88, got %q", resp.Username)
	}
	if resp.Claims == nil || resp.Claims["tenant_id"] != "7" || resp.Claims["sid"] != "sid-1" {
		t.Fatalf("expected private claims tenant_id/sid, got %v", resp.Claims)
	}
}

// TestSetUserinfoFromTokenHonorsTokenScopes 覆盖 M2：
// userinfo 只能返回 token 实际授权 scope 内的声明。
func TestSetUserinfoFromTokenHonorsTokenScopes(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ps, db := newProtocolConformanceStore(t)
	seedPerson(t, db, "88", "person@example.com")

	// 仅 openid：不得返回 email/name
	storeAccessTokenMeta(context.Background(), "at-only-openid", accessTokenMeta{
		Subject:   buildOIDCSubject("88"),
		ClientID:  "client-1",
		Scopes:    []string{oidc.ScopeOpenID},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	})
	info := &oidc.UserInfo{}
	if err := ps.SetUserinfoFromToken(context.Background(), info, "at-only-openid", buildOIDCSubject("88"), ""); err != nil {
		t.Fatalf("SetUserinfoFromToken failed: %v", err)
	}
	if info.Email != "" || info.Name != "" {
		t.Fatalf("expected no userinfo claims for openid-only token, got email=%q name=%q", info.Email, info.Name)
	}

	// openid + email：返回 email，且 email_verified 必须为 false（H5，无验证流程不得宣称已验证）
	storeAccessTokenMeta(context.Background(), "at-with-email", accessTokenMeta{
		Subject:   buildOIDCSubject("88"),
		ClientID:  "client-1",
		Scopes:    []string{oidc.ScopeOpenID, oidc.ScopeEmail},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	})
	info = &oidc.UserInfo{}
	if err := ps.SetUserinfoFromToken(context.Background(), info, "at-with-email", buildOIDCSubject("88"), ""); err != nil {
		t.Fatalf("SetUserinfoFromToken failed: %v", err)
	}
	if info.Email != "person@example.com" {
		t.Fatalf("expected email person@example.com, got %q", info.Email)
	}
	if info.EmailVerified {
		t.Fatal("expected email_verified=false: no verification flow exists (H5)")
	}
}

// TestSetUserinfoFromScopesEmailVerifiedFalse 覆盖 H5：
// SetUserinfoFromScopes 同样不得宣称 email 已验证。
func TestSetUserinfoFromScopesEmailVerifiedFalse(t *testing.T) {
	ps, db := newProtocolConformanceStore(t)
	seedPerson(t, db, "88", "person@example.com")

	info := &oidc.UserInfo{}
	if err := ps.SetUserinfoFromScopes(context.Background(), info, buildOIDCSubject("88"), "client-1", []string{oidc.ScopeOpenID, oidc.ScopeEmail}); err != nil {
		t.Fatalf("SetUserinfoFromScopes failed: %v", err)
	}
	if info.Email != "person@example.com" {
		t.Fatalf("expected email person@example.com, got %q", info.Email)
	}
	if info.EmailVerified {
		t.Fatal("expected email_verified=false (H5)")
	}
}

// TestCreateAuthRequestEnforcesRequirePKCE 覆盖 M3：
// require_pkce=1 的客户端必须携带 code_challenge，否则授权请求被拒绝。
func TestCreateAuthRequestEnforcesRequirePKCE(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ps, db := newProtocolConformanceStore(t)
	if err := db.Create(&model.ApplicationClientEntity{
		BaseEntity:              gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}},
		Code:                   "pkce-client",
		RequirePKCE:             true,
		RedirectURIs:            datatypes.JSON(`["https://client.example.com/callback"]`),
		PostLogoutRedirectURIs:  datatypes.JSON("[]"),
		GrantTypes:              datatypes.JSON(`["authorization_code"]`),
		ResponseTypes:           datatypes.JSON(`["code"]`),
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodBasic,
		AllowedOrigins:          datatypes.JSON("[]"),
		DefaultScopes:           datatypes.JSON(`["openid"]`),
		Status:                  model.ApplicationClientStatusEnable,
		Type:                    model.ApplicationClientTypeFirstParty,
	}).Error; err != nil {
		t.Fatalf("seed pkce client: %v", err)
	}

	storage := NewOIDCStorage(NewRedisProtocolStateStore(), ps, nil, "test-key")

	// 无 code_challenge → 拒绝
	_, err := storage.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "pkce-client",
		RedirectURI:  "https://client.example.com/callback",
		Scopes:       []string{oidc.ScopeOpenID},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err == nil {
		t.Fatal("expected CreateAuthRequest to reject missing code_challenge for require_pkce client")
	}

	// 携带 S256 code_challenge → 通过
	req, err := storage.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:            "pkce-client",
		RedirectURI:         "https://client.example.com/callback",
		Scopes:              []string{oidc.ScopeOpenID},
		ResponseType:        oidc.ResponseTypeCode,
		ResponseMode:        oidc.ResponseModeQuery,
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: oidc.CodeChallengeMethodS256,
	}, "")
	if err != nil {
		t.Fatalf("expected CreateAuthRequest with PKCE to pass, got %v", err)
	}
	if req == nil {
		t.Fatal("expected non-nil auth request")
	}
}

// TestProtocolStoreRejectsPlainCodeChallenge 覆盖 M3：
// discovery 只声明 S256，plain code_challenge_method 必须被拒绝（防降级）。
func TestProtocolStoreRejectsPlainCodeChallenge(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	store := NewRedisProtocolStateStore()
	_, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:            "client-1",
		RedirectURI:         "https://client.example.com/callback",
		Scopes:              []string{oidc.ScopeOpenID},
		ResponseType:        oidc.ResponseTypeCode,
		ResponseMode:        oidc.ResponseModeQuery,
		CodeChallenge:       "plain-challenge-value",
		CodeChallengeMethod: "plain",
	}, "", "")
	if err == nil {
		t.Fatal("expected plain code_challenge_method to be rejected")
	}
}

// TestSetUserinfoFromRequestInjectsSid 覆盖 L8（M4 收尾）：
// ID token 签发时必须注入 sid 会话声明，供 id_token_hint 做会话粒度登出。
func TestSetUserinfoFromRequestInjectsSid(t *testing.T) {
	storage := NewOIDCStorage(nil, nil, nil, "")

	ar := &AuthRequest{SessionID: "sid-abc"}
	userinfo := &oidc.UserInfo{}
	if err := storage.SetUserinfoFromRequest(context.Background(), userinfo, ar, []string{oidc.ScopeOpenID}); err != nil {
		t.Fatalf("SetUserinfoFromRequest failed: %v", err)
	}
	if userinfo.Claims == nil || userinfo.Claims["sid"] != "sid-abc" {
		t.Fatalf("expected sid claim in ID token userinfo, got %v", userinfo.Claims)
	}

	// 无会话时不注入
	empty := &oidc.UserInfo{}
	_ = storage.SetUserinfoFromRequest(context.Background(), empty, &AuthRequest{}, []string{oidc.ScopeOpenID})
	if _, exists := empty.Claims["sid"]; exists {
		t.Fatal("expected no sid claim when session is empty")
	}
}

// TestLoadKeysFailClosedInNonDev 覆盖 H4：
// 非 dev 环境未配置签名/加密密钥时必须启动失败（fail-closed），禁止使用临时/测试密钥。
func TestLoadKeysFailClosedInNonDev(t *testing.T) {
	prev := appconfig.Conf
	defer func() { appconfig.Conf = prev }()
	appconfig.Conf = &pkgconfig.Config{
		Server: pkgconfig.Server{Env: "prod"},
		OIDC:   pkgconfig.OIDC{},
	}

	if _, _, err := loadSigningKey(); err == nil {
		t.Fatal("expected loadSigningKey to fail in non-dev without configured key")
	}
	if _, _, err := loadEncryptionKey(); err == nil {
		t.Fatal("expected loadEncryptionKey to fail in non-dev without configured key")
	}

	// dev 环境允许临时/测试密钥
	appconfig.Conf = &pkgconfig.Config{
		Server: pkgconfig.Server{Env: "dev"},
		OIDC:   pkgconfig.OIDC{},
	}
	if _, _, err := loadSigningKey(); err != nil {
		t.Fatalf("expected loadSigningKey to succeed in dev, got %v", err)
	}
	if _, _, err := loadEncryptionKey(); err != nil {
		t.Fatalf("expected loadEncryptionKey to succeed in dev, got %v", err)
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
