package svcoidc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLookupApiKeyByRawKey(t *testing.T) {
	dsn := fmt.Sprintf("file:api_key_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApiKeyEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	store := NewPersistentStore()
	store.apiKeyDao = func(opts ...dao.DaoOption) *dao.ApiKeyDao {
		return dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}

	sum := sha256.Sum256([]byte("rawkey"))
	hash := hex.EncodeToString(sum[:])

	if err := db.Create(&model.ApiKeyEntity{
		TenantID:  1,
		Name:      "test-key",
		KeyHash:   hash,
		KeyPrefix: "ak_12345",
		Scope:     json.RawMessage("{}"),
		CreatedBy: 7,
	}).Error; err != nil {
		t.Fatalf("create api key: %v", err)
	}

	entity, err := store.LookupApiKeyByRawKey(context.Background(), "rawkey")
	if err != nil {
		t.Fatalf("LookupApiKeyByRawKey returned error: %v", err)
	}
	if entity == nil {
		t.Fatal("expected to find api key, got nil")
	}
	if entity.TenantID != 1 {
		t.Fatalf("expected TenantID 1, got %d", entity.TenantID)
	}
	if entity.CreatedBy != 7 {
		t.Fatalf("expected CreatedBy 7, got %d", entity.CreatedBy)
	}

	miss, err := store.LookupApiKeyByRawKey(context.Background(), "wrong")
	if err != nil {
		t.Fatalf("LookupApiKeyByRawKey (miss) returned error: %v", err)
	}
	if miss != nil {
		t.Fatalf("expected nil for unknown raw key, got %+v", miss)
	}
}

func TestClientCredentialsForApiKey(t *testing.T) {
	ctx := context.Background()

	dsn := fmt.Sprintf("file:api_key_cc_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApiKeyEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.AutoMigrate(&model.UserEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	sum := sha256.Sum256([]byte("k1"))
	hash := hex.EncodeToString(sum[:])
	if err := db.Create(&model.ApiKeyEntity{
		TenantID:  1,
		Name:      "service-key",
		KeyHash:   hash,
		KeyPrefix: "ak_1234567",
		Scope:     json.RawMessage("{}"),
		CreatedBy: 7,
	}).Error; err != nil {
		t.Fatalf("create api key: %v", err)
	}
	// owner user：ID=7（对应 apiKey.CreatedBy），person_id=5
	now := time.Now()
	ownerUser := &model.UserEntity{
		TenantID:   1,
		PersonID:   5,
		Profile:    json.RawMessage("{}"),
		CustomData: json.RawMessage("{}"),
		JoinedAt:   &now,
	}
	ownerUser.ID = 7
	if err := db.Create(ownerUser).Error; err != nil {
		t.Fatalf("create owner user: %v", err)
	}

	persistentStore := NewPersistentStore()
	persistentStore.apiKeyDao = func(opts ...dao.DaoOption) *dao.ApiKeyDao {
		return dao.NewApiKeyDao(dao.WithDBGetter(func(c context.Context) *gorm.DB { return db.WithContext(c) }))
	}
	persistentStore.userDao = func(opts ...dao.DaoOption) *dao.UserDao {
		return &dao.UserDao{Dao: gormdao.NewDao[model.UserEntity, model.UserEntityList, uint](
			model.TableNameUser, "UserDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}
	persistentStore.applicationClientDao = func(opts ...dao.DaoOption) *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(func(c context.Context) *gorm.DB { return db.WithContext(c) }))
	}

	storage := NewOIDCStorage(nil, persistentStore, nil, "test-key")

	client, err := storage.ClientCredentials(ctx, "k1", "k1")
	if err != nil {
		t.Fatalf("ClientCredentials with API key credentials failed: %v", err)
	}
	if client.GetID() != "ak_1234567" {
		t.Fatalf("expected api key client id %q, got %q", "ak_1234567", client.GetID())
	}

	_, err = storage.ClientCredentials(ctx, "k1", "wrong")
	if err == nil {
		t.Fatal("expected ClientCredentials with wrong secret to fail")
	}
	var oidcErr *oidc.Error
	if !errors.As(err, &oidcErr) || oidcErr.ErrorType != oidc.ErrInvalidClient().ErrorType {
		t.Fatalf("expected oidc.ErrInvalidClient on wrong secret, got %v", err)
	}

	req, err := storage.ClientCredentialsTokenRequest(ctx, "k1", []string{"urn:ark:iam:admin"})
	if err != nil {
		t.Fatalf("ClientCredentialsTokenRequest failed: %v", err)
	}
	if _, ok := req.(*clientCredentialsTokenRequest); !ok {
		t.Fatalf("expected *clientCredentialsTokenRequest, got %T", req)
	}
	// H1：sub/aud/client_id 一律用 key 前缀，绝不把原始 API Key 写进 token claim
	if got := req.GetSubject(); got != "ak_1234567" {
		t.Fatalf("expected subject %q, got %q", "ak_1234567", got)
	}
	if got := req.GetAudience(); len(got) != 1 || got[0] != "ak_1234567" {
		t.Fatalf("expected audience [ak_1234567], got %v", got)
	}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, req, []string{"urn:ark:iam:admin"})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got := claims["tenant_id"]; got != uint(1) {
		t.Fatalf("expected tenant_id claim 1, got %v", got)
	}
	if got := claims["user_id"]; got != uint(7) {
		t.Fatalf("expected user_id claim 7, got %v", got)
	}
	if got := claims["client_id"]; got != "ak_1234567" {
		t.Fatalf("expected client_id claim %q, got %v", "ak_1234567", got)
	}
	if got := claims["token_usage"]; got != "machine" {
		t.Fatalf("expected token_usage claim machine, got %v", got)
	}
}
