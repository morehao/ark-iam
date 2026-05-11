package dao

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/morehao/ark-iam/iam/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApiKeyDaoInsert(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	dao := NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	rawKey := "test-key-raw-value-1234567890"
	keyHash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate key hash: %v", err)
	}

	entity := &model.ApiKeyEntity{
		TenantID:  1,
		Name:      "Test Key",
		KeyHash:   string(keyHash),
		KeyPrefix: rawKey[:7],
		Scope:     json.RawMessage(`{"read":true}`),
		CreatedBy: 1,
	}

	if err := dao.Insert(t.Context(), entity); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if entity.ID == 0 {
		t.Fatal("expected non-zero ID after insert")
	}
}

func TestApiKeyDaoGetByID(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	dao := NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	rawKey := "test-key-for-get-by-id"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	entity := &model.ApiKeyEntity{
		TenantID:  1,
		Name:      "GetByID Key",
		KeyHash:   string(keyHash),
		KeyPrefix: rawKey[:7],
		Scope:     json.RawMessage(`{}`),
		CreatedBy: 1,
	}
	if err := dao.Insert(t.Context(), entity); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	result, err := dao.GetByID(t.Context(), entity.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != "GetByID Key" {
		t.Fatalf("expected name 'GetByID Key', got '%s'", result.Name)
	}
	if result.KeyPrefix != "test-ke" {
		t.Fatalf("expected keyPrefix 'test-ke', got '%s'", result.KeyPrefix)
	}
}

func TestApiKeyDaoGetPageListByCond(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	dao := NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	revokedTime := time.Now()

	entities := []*model.ApiKeyEntity{
		{TenantID: 1, Name: "Key A", KeyHash: "hash-a", KeyPrefix: "prefixA", CreatedBy: 1, Scope: json.RawMessage(`{}`)},
		{TenantID: 1, Name: "Key B", KeyHash: "hash-b", KeyPrefix: "prefixB", CreatedBy: 1, Scope: json.RawMessage(`{}`)},
		{TenantID: 1, Name: "Key C", KeyHash: "hash-c", KeyPrefix: "prefixC", CreatedBy: 1, RevokedAt: &revokedTime, Scope: json.RawMessage(`{}`)},
		{TenantID: 2, Name: "Other Tenant", KeyHash: "hash-d", KeyPrefix: "prefixD", CreatedBy: 2, Scope: json.RawMessage(`{}`)},
	}

	for _, e := range entities {
		entity := e
		if err := dao.Insert(t.Context(), entity); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	t.Run("filter by tenant_id", func(t *testing.T) {
		cond := &ApiKeyCond{TenantID: 1}
		list, total, err := dao.GetPageListByCond(t.Context(), cond, 1, 10)
		if err != nil {
			t.Fatalf("GetPageListByCond failed: %v", err)
		}
		if total != 3 {
			t.Fatalf("expected 3 results for tenant 1, got %d", total)
		}
		_ = list
	})

	t.Run("filter by name", func(t *testing.T) {
		cond := &ApiKeyCond{TenantID: 1, Name: "Key B"}
		list, total, err := dao.GetPageListByCond(t.Context(), cond, 1, 10)
		if err != nil {
			t.Fatalf("GetPageListByCond failed: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected 1 result for name 'Key B', got %d", total)
		}
		if len(list) > 0 && list[0].Name != "Key B" {
			t.Fatalf("expected name 'Key B', got '%s'", list[0].Name)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		cond := &ApiKeyCond{TenantID: 1}
		list, total, err := dao.GetPageListByCond(t.Context(), cond, 1, 2)
		if err != nil {
			t.Fatalf("GetPageListByCond failed: %v", err)
		}
		if total != 3 {
			t.Fatalf("expected total 3, got %d", total)
		}
		if len(list) != 2 {
			t.Fatalf("expected page 1 to have 2 results, got %d", len(list))
		}
	})
}

func TestApiKeyDaoUpdateLastUsedAt(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	dao := NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	entity := &model.ApiKeyEntity{
		TenantID:  1,
		Name:      "Usage Update Key",
		KeyHash:   "hash-usage",
		KeyPrefix: "usage-p",
		CreatedBy: 1,
		Scope:     json.RawMessage(`{}`),
	}
	if err := dao.Insert(t.Context(), entity); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if err := dao.UpdateLastUsedAt(t.Context(), entity.ID); err != nil {
		t.Fatalf("UpdateLastUsedAt failed: %v", err)
	}

	result, err := dao.GetByID(t.Context(), entity.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if !result.LastUsedAt.Valid {
		t.Fatal("expected LastUsedAt to be set")
	}
}

func TestApiKeyDaoRevoke(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	dao := NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	entity := &model.ApiKeyEntity{
		TenantID:  1,
		Name:      "Revoke Key",
		KeyHash:   "hash-revoke",
		KeyPrefix: "revoke-",
		CreatedBy: 1,
		Scope:     json.RawMessage(`{}`),
	}
	if err := dao.Insert(t.Context(), entity); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if err := dao.Revoke(t.Context(), entity.ID); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	result, err := dao.GetByID(t.Context(), entity.ID)
	if err != nil {
		t.Fatalf("GetByID after revoke failed: %v", err)
	}
	if result.RevokedAt == nil {
		t.Fatal("expected RevokedAt to be set after revoke")
	}
}

func TestApiKeyDaoDelete(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	dao := NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	entity := &model.ApiKeyEntity{
		TenantID:  1,
		Name:      "Delete Key",
		KeyHash:   "hash-delete",
		KeyPrefix: "delete-",
		CreatedBy: 1,
		Scope:     json.RawMessage(`{}`),
	}
	if err := dao.Insert(t.Context(), entity); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if err := dao.Delete(t.Context(), entity.ID, 1); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	result, err := dao.GetByID(t.Context(), entity.ID)
	if err != nil {
		t.Fatalf("GetByID after delete failed: %v", err)
	}
	// After Delete (soft delete via GORM), GORM will exclude it from normal queries
	// Our GetByID doesn't use Unscoped, so it should return nil
	if result != nil {
		// If we got a result, check that it has deleted_at set
		if !result.DeletedAt.Valid {
			t.Fatal("expected deleted record to have non-nil deletedAt")
		}
	} else {
		// nil is also OK - GORM excluded it
	}
}

func setupApiKeyTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.ApiKeyEntity{}); err != nil {
		t.Fatalf("migrate api_key: %v", err)
	}
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
	return db, cleanup
}
