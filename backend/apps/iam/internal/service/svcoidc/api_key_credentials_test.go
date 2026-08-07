package svcoidc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
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
	store.apiKeyDao = func() *dao.ApiKeyDao {
		return dao.NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) })
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
