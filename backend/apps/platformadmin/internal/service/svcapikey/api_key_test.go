package svcapikey

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapikey"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateApiKey(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.ApiKeyCreateReq{
		Name: "My Test Key",
	}
	resp, err := svc.Create(newTestGinCtx("1"), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.ID == "" {
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
	ctx := newTestGinCtxWithUser("1", "7")

	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.ApiKeyCreateReq{
		Name: "Owner-bound Key",
	}
	resp, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var entity model.ApiKeyEntity
	if err := db.First(&entity, "id = ?", resp.ID).Error; err != nil {
		t.Fatalf("query persisted entity failed: %v", err)
	}
	if entity.CreatedBy != "7" {
		t.Fatalf("expected CreatedBy=7, got %s", entity.CreatedBy)
	}
	if entity.TenantID != "1" {
		t.Fatalf("expected TenantID=1, got %s", entity.TenantID)
	}
}

func TestCreateApiKeyReturnsKeyOnlyOnce(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.ApiKeyCreateReq{Name: "One-time Key"}
	resp, err := svc.Create(newTestGinCtx("1"), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	key := resp.Key

	pageResp, err := svc.PageList(newTestGinCtx("1"), &dtoapikey.ApiKeyPageListReq{Name: "One-time Key"})
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

	req := &dtoapikey.ApiKeyCreateReq{Name: "Revokable Key"}
	resp, err := svc.Create(newTestGinCtx("1"), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Revoke(newTestGinCtx("1"), &dtoapikey.RevokeApiKeyReq{ApiKeyID: resp.ID}); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	pageResp, err := svc.PageList(newTestGinCtx("1"), &dtoapikey.ApiKeyPageListReq{Name: "Revokable Key"})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(pageResp.List) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if pageResp.List[0].RevokedAt == 0 {
		t.Fatal("expected RevokedAt to be set")
	}
}

func TestDeleteApiKey(t *testing.T) {
	svc, db, cleanup := newTestApiKeySvc(t)
	defer cleanup()

	_ = db.AutoMigrate(&model.ApiKeyEntity{})

	req := &dtoapikey.ApiKeyCreateReq{Name: "Deletable Key"}
	resp, err := svc.Create(newTestGinCtx("1"), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(newTestGinCtx("1"), &dtoapikey.ApiKeyDeleteReq{ApiKeyID: resp.ID}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	pageResp, err := svc.PageList(newTestGinCtx("1"), &dtoapikey.ApiKeyPageListReq{Name: "Deletable Key"})
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
		req := &dtoapikey.ApiKeyCreateReq{Name: "Batch Key"}
		_, err := svc.Create(newTestGinCtx("1"), req)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	pageResp, err := svc.PageList(newTestGinCtx("1"), &dtoapikey.ApiKeyPageListReq{Name: "Batch Key"})
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

	req1 := &dtoapikey.ApiKeyCreateReq{Name: "Tenant A Key"}
	resp1, err := svc.Create(newTestGinCtx("1"), req1)
	if err != nil {
		t.Fatalf("Create tenant 1 failed: %v", err)
	}

	req2 := &dtoapikey.ApiKeyCreateReq{Name: "Tenant B Key"}
	_, err = svc.Create(newTestGinCtx("2"), req2)
	if err != nil {
		t.Fatalf("Create tenant 2 failed: %v", err)
	}

	pageResp1, err := svc.PageList(newTestGinCtx("1"), &dtoapikey.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("PageList tenant 1 failed: %v", err)
	}

	pageResp2, err := svc.PageList(newTestGinCtx("2"), &dtoapikey.ApiKeyPageListReq{})
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

func TestPageListSupervisionCrossTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApiKeyEntity{}, &model.UserEntity{}, &model.TenantEntity{})
	svc := NewCreateApiKeySvc() // 默认 dao getter → 落到 SetupSQLite 注册的全局 iam 测试库

	tenantA := &model.TenantEntity{Code: "tenant-a", Name: "租户A"}
	tenantB := &model.TenantEntity{Code: "tenant-b", Name: "租户B"}
	if err := db.Create(tenantA).Error; err != nil {
		t.Fatalf("seed tenantA: %v", err)
	}
	if err := db.Create(tenantB).Error; err != nil {
		t.Fatalf("seed tenantB: %v", err)
	}
	userA := &model.UserEntity{TenantID: tenantA.ID, Name: "A管理员", Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`)}
	userB := &model.UserEntity{TenantID: tenantB.ID, Name: "B管理员", Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`)}
	if err := db.Create(userA).Error; err != nil {
		t.Fatalf("seed userA: %v", err)
	}
	if err := db.Create(userB).Error; err != nil {
		t.Fatalf("seed userB: %v", err)
	}

	keyA, err := svc.Create(newTestGinCtxWithUser(tenantA.ID, userA.ID), &dtoapikey.ApiKeyCreateReq{Name: "A-Key"})
	if err != nil {
		t.Fatalf("Create tenantA key: %v", err)
	}
	if _, err := svc.Create(newTestGinCtxWithUser(tenantB.ID, userB.ID), &dtoapikey.ApiKeyCreateReq{Name: "B-Key"}); err != nil {
		t.Fatalf("Create tenantB key: %v", err)
	}

	// 普通列表仍按上下文租户隔离
	pageResp, err := svc.PageList(newTestGinCtx(tenantA.ID), &dtoapikey.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("PageList: %v", err)
	}
	if len(pageResp.List) != 1 || pageResp.List[0].ID != keyA.ID {
		t.Fatalf("tenant-scoped list should contain only tenantA key, got %+v", pageResp.List)
	}

	// 监督视图忽略上下文租户：两台 key 都能看到，且归属名被解析
	sup, err := svc.PageListSupervision(newTestGinCtx(tenantA.ID), &dtoapikey.ApiKeySupervisionPageListReq{})
	if err != nil {
		t.Fatalf("PageListSupervision: %v", err)
	}
	if sup.Total != 2 || len(sup.List) != 2 {
		t.Fatalf("supervision should list keys of both tenants, got total=%d list=%d", sup.Total, len(sup.List))
	}
	byName := map[string]dtoapikey.ApiKeySupervisionItem{}
	for _, item := range sup.List {
		byName[item.Name] = item
	}
	for _, item := range []dtoapikey.ApiKeySupervisionItem{byName["A-Key"], byName["B-Key"]} {
		if item.TenantName == "" || item.CreatorName == "" {
			t.Fatalf("supervision item missing owner names: %+v", item)
		}
	}
	if byName["A-Key"].TenantName != "租户A" || byName["B-Key"].TenantName != "租户B" {
		t.Fatalf("unexpected tenant names: %+v", byName)
	}

	// 支持按租户过滤（可选查询参数）
	supFiltered, err := svc.PageListSupervision(newTestGinCtx(tenantA.ID), &dtoapikey.ApiKeySupervisionPageListReq{TenantID: tenantB.ID})
	if err != nil {
		t.Fatalf("PageListSupervision filtered: %v", err)
	}
	if supFiltered.Total != 1 || supFiltered.List[0].Name != "B-Key" {
		t.Fatalf("filtered supervision mismatch: %+v", supFiltered.List)
	}
}

func newTestApiKeySvc(t *testing.T) (CreateApiKeySvc, *gorm.DB, func()) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	apiKeyDao := dao.NewApiKeyDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	svc := newCreateApiKeySvcWithDao(apiKeyDao)

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	return svc, db, cleanup
}

func newTestGinCtx(tenantID string) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set(gcontext.KeyTenantID, tenantID)
	return ctx
}

func newTestGinCtxWithUser(tenantID, userID string) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

func sanitizeTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}
