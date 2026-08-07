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

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
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
	if err := db.AutoMigrate(&model.OAuthClientEntity{}, &model.OAuthClientSecretEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	clientID := "machine-client"
	secret := "s3cr3t"
	secretHash := sha256.Sum256([]byte(secret))

	clientEntity := &model.OAuthClientEntity{
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
		Status:                  model.OAuthClientStatusEnable,
		Type:                    model.OAuthClientTypeFirstParty,
	}
	if err := db.Create(clientEntity).Error; err != nil {
		t.Fatalf("insert oauth_client: %v", err)
	}

	secretEntity := &model.OAuthClientSecretEntity{
		Model:         gorm.Model{ID: 1},
		OAuthClientID: clientEntity.ID,
		Name:          "default",
		ValueHash:     hex.EncodeToString(secretHash[:]),
		ValuePrefix:   "s*",
	}
	if err := db.Create(secretEntity).Error; err != nil {
		t.Fatalf("insert oauth_client_secret: %v", err)
	}

	persistentStore := NewPersistentStore()
	persistentStore.oauthClientDao = func() *dao.OAuthClientDao {
		return dao.NewOAuthClientDaoWithDB(func(c context.Context) *gorm.DB { return db.WithContext(c) })
	}
	persistentStore.oauthClientSecretDao = func() *dao.OAuthClientSecretDao {
		return &dao.OAuthClientSecretDao{Dao: gormdao.NewDao[model.OAuthClientSecretEntity, model.OAuthClientSecretEntityList](
			model.TableNameOAuthClientSecret, "OAuthClientSecretDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
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
