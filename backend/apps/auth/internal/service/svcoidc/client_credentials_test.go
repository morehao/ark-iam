package svcoidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestClientCredentialsStorage(t *testing.T) {
	ctx := context.Background()

	dsn := fmt.Sprintf("file:cc_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	clientID := "machine-client"
	secret := "s3cr3t"
	secretHash := sha256.Sum256([]byte(secret))

	clientEntity := &model.ApplicationClientEntity{
		Model:                   gorm.Model{ID: 1},
		TenantID:                1,
		ClientID:                clientID,
		Name:                    "Machine Client",
		RedirectURIs:            datatypes.JSON("[]"),
		PostLogoutRedirectURIs:  datatypes.JSON("[]"),
		GrantTypes:              datatypes.JSON(fmt.Sprintf(`["%s"]`, model.GrantTypeClientCredentials)),
		ResponseTypes:           datatypes.JSON("[]"),
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodBasic,
		AllowedOrigins:          datatypes.JSON("[]"),
		DefaultScopes:           datatypes.JSON(`["openid"]`),
		Status:                  model.ApplicationClientStatusEnable,
		Type:                    model.ApplicationClientTypeFirstParty,
	}
	if err := db.Create(clientEntity).Error; err != nil {
		t.Fatalf("insert application_client: %v", err)
	}

	secretEntity := &model.ApplicationClientSecretEntity{
		Model:               gorm.Model{ID: 1},
		ApplicationClientID: clientEntity.ID,
		Name:                "default",
		ValueHash:           hex.EncodeToString(secretHash[:]),
		ValuePrefix:         "s*",
	}
	if err := db.Create(secretEntity).Error; err != nil {
		t.Fatalf("insert application_client_secret: %v", err)
	}

	persistentStore := NewPersistentStore()
	persistentStore.applicationClientDao = func() *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(func(c context.Context) *gorm.DB { return db.WithContext(c) }))
	}
	persistentStore.applicationClientSecretDao = func() *dao.ApplicationClientSecretDao {
		return &dao.ApplicationClientSecretDao{Dao: gormdao.NewDao[model.ApplicationClientSecretEntity, model.ApplicationClientSecretEntityList](
			model.TableNameApplicationClientSecret, "ApplicationClientSecretDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}
	persistentStore.apiKeyDao = func() *dao.ApiKeyDao {
		return dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	storage := NewOIDCStorage(nil, persistentStore, privateKey, "test-key")

	ccStorage, ok := (interface{})(storage).(op.ClientCredentialsStorage)
	if !ok {
		t.Fatal("expected OIDCStorage to implement op.ClientCredentialsStorage")
	}

	client, err := ccStorage.ClientCredentials(ctx, clientID, secret)
	if err != nil {
		t.Fatalf("ClientCredentials with correct secret failed: %v", err)
	}
	if client.GetID() != clientID {
		t.Fatalf("expected client id %q, got %q", clientID, client.GetID())
	}

	_, err = ccStorage.ClientCredentials(ctx, clientID, "wrong-secret")
	if err == nil {
		t.Fatal("expected ClientCredentials with wrong secret to fail")
	}
	var oidcErr *oidc.Error
	if !errors.As(err, &oidcErr) || oidcErr.ErrorType != oidc.ErrInvalidClient().ErrorType {
		t.Fatalf("expected oidc.ErrInvalidClient on wrong secret, got %v", err)
	}

	req, err := ccStorage.ClientCredentialsTokenRequest(ctx, clientID, []string{oidc.ScopeOpenID})
	if err != nil {
		t.Fatalf("ClientCredentialsTokenRequest failed: %v", err)
	}
	if req.GetSubject() != clientID {
		t.Fatalf("expected subject %q, got %q", clientID, req.GetSubject())
	}
}

func TestClientCredentialsRejectsPublicClient(t *testing.T) {
	ctx := context.Background()

	dsn := fmt.Sprintf("file:cc_public_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	clientID := "public-client"
	secret := "s3cr3t"
	secretHash := sha256.Sum256([]byte(secret))

	publicClient := &model.ApplicationClientEntity{
		Model:                   gorm.Model{ID: 1},
		TenantID:                1,
		ClientID:                clientID,
		Name:                    "Public Client",
		RedirectURIs:            datatypes.JSON("[]"),
		PostLogoutRedirectURIs:  datatypes.JSON("[]"),
		GrantTypes:              datatypes.JSON(fmt.Sprintf(`["%s"]`, model.GrantTypeClientCredentials)),
		ResponseTypes:           datatypes.JSON("[]"),
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodNone,
		AllowedOrigins:          datatypes.JSON("[]"),
		DefaultScopes:           datatypes.JSON(`["openid"]`),
		Status:                  model.ApplicationClientStatusEnable,
		Type:                    model.ApplicationClientTypeFirstParty,
	}
	if err := db.Create(publicClient).Error; err != nil {
		t.Fatalf("insert application_client: %v", err)
	}

	secretEntity := &model.ApplicationClientSecretEntity{
		Model:               gorm.Model{ID: 1},
		ApplicationClientID: publicClient.ID,
		Name:                "default",
		ValueHash:           hex.EncodeToString(secretHash[:]),
		ValuePrefix:         "s*",
	}
	if err := db.Create(secretEntity).Error; err != nil {
		t.Fatalf("insert application_client_secret: %v", err)
	}

	persistentStore := NewPersistentStore()
	persistentStore.applicationClientDao = func() *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(func(c context.Context) *gorm.DB { return db.WithContext(c) }))
	}
	persistentStore.applicationClientSecretDao = func() *dao.ApplicationClientSecretDao {
		return &dao.ApplicationClientSecretDao{Dao: gormdao.NewDao[model.ApplicationClientSecretEntity, model.ApplicationClientSecretEntityList](
			model.TableNameApplicationClientSecret, "ApplicationClientSecretDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}
	persistentStore.apiKeyDao = func() *dao.ApiKeyDao {
		return dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}

	storage := NewOIDCStorage(nil, persistentStore, nil, "test-key")

	ccStorage, ok := (interface{})(storage).(op.ClientCredentialsStorage)
	if !ok {
		t.Fatal("expected OIDCStorage to implement op.ClientCredentialsStorage")
	}

	if client, err := ccStorage.ClientCredentials(ctx, clientID, secret); err == nil {
		t.Fatalf("expected public client (%s) with auth method none to be rejected, got client %q", clientID, client.GetID())
	}
}

func TestCreateAccessTokenForClientCredentials(t *testing.T) {
	ctx := context.Background()

	clientID := "cc-client-ttl"
	accessTokenTTL := int64(7200)
	db := newClientCredentialsTestDB(t, clientID, accessTokenTTL)

	persistentStore := NewPersistentStore()
	persistentStore.applicationClientDao = func() *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(func(c context.Context) *gorm.DB { return db.WithContext(c) }))
	}

	now := time.Now()
	accessTokenID, expiration, err := persistentStore.CreateAccessToken(ctx, &clientCredentialsTokenRequest{
		subject:  clientID,
		audience: []string{"urn:ark:iam:admin"},
		clientID: clientID,
		scopes:   []string{"openid"},
	})
	if err != nil {
		t.Fatalf("CreateAccessToken failed: %v", err)
	}
	if accessTokenID == "" {
		t.Fatal("expected non-empty access token id")
	}
	if !expiration.After(now) {
		t.Fatalf("expected expiration in the future, got %v", expiration)
	}
	ttlSeconds := int64(expiration.Sub(now).Seconds())
	if ttlSeconds != accessTokenTTL {
		t.Fatalf("expected access token ttl to match client's %d, got %d", accessTokenTTL, ttlSeconds)
	}
}

func TestGetPrivateClaimsFromRequestForClientCredentials(t *testing.T) {
	ctx := context.Background()

	clientID := "cc-client-private"
	db := newClientCredentialsTestDB(t, clientID, 3600)

	persistentStore := NewPersistentStore()
	persistentStore.applicationClientDao = func() *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(func(c context.Context) *gorm.DB { return db.WithContext(c) }))
	}

	storage := NewOIDCStorage(nil, persistentStore, nil, "test-key")

	req := &clientCredentialsTokenRequest{
		subject:  clientID,
		clientID: clientID,
		audience: []string{"urn:ark:iam:admin"},
		scopes:   []string{"urn:ark:iam:admin"},
	}
	claims, err := storage.GetPrivateClaimsFromRequest(ctx, req, []string{"urn:ark:iam:admin"})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got := claims["client_id"]; got != clientID {
		t.Fatalf("expected client_id claim %q, got %v", clientID, got)
	}
}

func TestResolveAudienceFromScopes(t *testing.T) {
	if got := resolveAudienceFromScopes([]string{"openid", "urn:ark:iam:admin"}); got != "urn:ark:iam:admin" {
		t.Fatalf("expected urn:ark:iam:admin, got %q", got)
	}
	if got := resolveAudienceFromScopes([]string{"openid", "profile", "email", "phone", "offline_access"}); got != "" {
		t.Fatalf("expected empty audience for standard scopes, got %q", got)
	}
}

func newClientCredentialsTestDB(t *testing.T, clientID string, accessTokenTTL int64) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:ccttl_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	clientEntity := &model.ApplicationClientEntity{
		Model:                   gorm.Model{ID: 1},
		TenantID:                1,
		ClientID:                clientID,
		Name:                    "TTL Client",
		RedirectURIs:            datatypes.JSON("[]"),
		PostLogoutRedirectURIs:  datatypes.JSON("[]"),
		GrantTypes:              datatypes.JSON(fmt.Sprintf(`["%s"]`, model.GrantTypeClientCredentials)),
		ResponseTypes:           datatypes.JSON("[]"),
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodBasic,
		AllowedOrigins:          datatypes.JSON("[]"),
		DefaultScopes:           datatypes.JSON(`["openid"]`),
		AccessTokenTTL:          accessTokenTTL,
		Status:                  model.ApplicationClientStatusEnable,
		Type:                    model.ApplicationClientTypeFirstParty,
	}
	if err := db.Create(clientEntity).Error; err != nil {
		t.Fatalf("insert application_client: %v", err)
	}
	return db
}
