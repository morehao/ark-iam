package svcapikey

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapikey"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateApiKey(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.CreateApiKeyReq{
		Name: "My Test Key",
	}
	resp, err := svc.Create(newTestGinCtx(1), 1, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if resp.Name != "My Test Key" {
		t.Fatalf("expected name 'My Test Key', got '%s'", resp.Name)
	}
	if resp.Key == "" {
		t.Fatal("expected non-empty key")
	}
	if len(resp.Key) != 64 {
		t.Fatalf("expected key length 64 (32 bytes hex), got %d", len(resp.Key))
	}
	if resp.KeyPrefix == "" {
		t.Fatal("expected non-empty keyPrefix")
	}
	if len(resp.KeyPrefix) != 7 {
		t.Fatalf("expected keyPrefix length 7, got %d", len(resp.KeyPrefix))
	}
}

func TestCreateApiKeyCapturesOwnerUser(t *testing.T) {
	ctx := newTestGinCtxWithUser(1, 7)

	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.CreateApiKeyReq{
		Name: "Owner-bound Key",
	}
	resp, err := svc.Create(ctx, 1, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var entity model.ApiKeyEntity
	if err := db.First(&entity, resp.ID).Error; err != nil {
		t.Fatalf("query persisted entity failed: %v", err)
	}
	if entity.CreatedBy != 7 {
		t.Fatalf("expected CreatedBy=7, got %d", entity.CreatedBy)
	}
	if entity.TenantID != 1 {
		t.Fatalf("expected TenantID=1, got %d", entity.TenantID)
	}
}

func TestCreateApiKeyReturnsKeyOnlyOnce(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.CreateApiKeyReq{Name: "One-time Key"}
	resp, err := svc.Create(newTestGinCtx(1), 1, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	key := resp.Key

	pageResp, err := svc.PageList(newTestGinCtx(1), 1, &dtoapikey.ApiKeyPageListReq{Name: "One-time Key"})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(pageResp.List) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if pageResp.List[0].KeyPrefix == "" {
		t.Fatal("expected keyPrefix in list")
	}
	if pageResp.List[0].KeyPrefix != key[:7] {
		t.Fatalf("expected keyPrefix '%s', got '%s'", key[:7], pageResp.List[0].KeyPrefix)
	}
}

func TestRevokeApiKey(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.CreateApiKeyReq{Name: "Revokable Key"}
	resp, err := svc.Create(newTestGinCtx(1), 1, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Revoke(newTestGinCtx(1), 1, &dtoapikey.RevokeApiKeyReq{ID: resp.ID}); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	pageResp, err := svc.PageList(newTestGinCtx(1), 1, &dtoapikey.ApiKeyPageListReq{Name: "Revokable Key"})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(pageResp.List) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if pageResp.List[0].RevokedAt == "" {
		t.Fatal("expected RevokedAt to be set")
	}
}

func TestDeleteApiKey(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.CreateApiKeyReq{Name: "Deletable Key"}
	resp, err := svc.Create(newTestGinCtx(1), 1, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(newTestGinCtx(1), 1, &dtoapikey.DeleteApiKeyReq{ID: resp.ID}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	pageResp, err := svc.PageList(newTestGinCtx(1), 1, &dtoapikey.ApiKeyPageListReq{Name: "Deletable Key"})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(pageResp.List) != 0 {
		t.Fatalf("expected 0 results after delete, got %d", len(pageResp.List))
	}
}

func TestPageListApiKey(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	for i := 0; i < 3; i++ {
		req := &dtoapikey.CreateApiKeyReq{Name: "Batch Key"}
		_, err := svc.Create(newTestGinCtx(1), 1, req)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	pageResp, err := svc.PageList(newTestGinCtx(1), 1, &dtoapikey.ApiKeyPageListReq{Name: "Batch Key"})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(pageResp.List) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if pageResp.Total == 0 {
		t.Fatal("expected non-zero total")
	}
	for _, item := range pageResp.List {
		if item.Name != "Batch Key" {
			t.Fatalf("expected name 'Batch Key', got '%s'", item.Name)
		}
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req1 := &dtoapikey.CreateApiKeyReq{Name: "Tenant A Key"}
	resp1, err := svc.Create(newTestGinCtx(1), 1, req1)
	if err != nil {
		t.Fatalf("Create tenant 1 failed: %v", err)
	}

	req2 := &dtoapikey.CreateApiKeyReq{Name: "Tenant B Key"}
	_, err = svc.Create(newTestGinCtx(2), 2, req2)
	if err != nil {
		t.Fatalf("Create tenant 2 failed: %v", err)
	}

	pageResp1, err := svc.PageList(newTestGinCtx(1), 1, &dtoapikey.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("PageList tenant 1 failed: %v", err)
	}

	pageResp2, err := svc.PageList(newTestGinCtx(2), 2, &dtoapikey.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("PageList tenant 2 failed: %v", err)
	}

	for _, item := range pageResp1.List {
		if item.ID == resp1.ID && item.Name == "Tenant A Key" {
			continue
		}
	}
	for _, item := range pageResp2.List {
		if item.Name == "Tenant A Key" {
			t.Fatal("Tenant B should not see Tenant A's keys")
		}
	}
	_ = resp1
}

func newTestApiKeySvc(t *testing.T) (CreateApiKeySvc, *gorm.DB, func()) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	apiKeyDao := dao.NewApiKeyDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	svc := newCreateApiKeySvcWithDao(apiKeyDao)

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	return svc, db, cleanup
}

func newTestGinCtx(tenantID uint) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set(gcontext.KeyTenantID, tenantID)
	return ctx
}

func newTestGinCtxWithUser(tenantID, userID uint) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

func sanitizeTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}
