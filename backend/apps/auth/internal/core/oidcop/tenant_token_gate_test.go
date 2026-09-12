package oidcop

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
)

// newTenantTokenGateTestStorage 构造只装配 tenant dao 的 OIDCStorage，用于租户准入门禁单测。
func newTenantTokenGateTestStorage(t *testing.T) (*OIDCStorage, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:tenant_token_gate_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.TenantEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	persistentStore := NewPersistentStore()
	persistentStore.tenantDao = func(opts ...dao.DaoOption) *dao.TenantDao {
		return dao.NewTenantDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }))
	}
	persistentStore.db = func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }
	return &OIDCStorage{persistentStore: persistentStore}, db
}

// TestTenantTokenGateRejectsInactiveTenant 令牌签发前的租户准入门禁：
// active 放行；suspended 拒绝并按调用方给定的 OAuth 错误类型返回；
// 租户行缺失属数据不一致，同样拒绝但描述必须区分于"挂起"。
func TestTenantTokenGateRejectsInactiveTenant(t *testing.T) {
	storage, db := newTenantTokenGateTestStorage(t)
	ctx := context.Background()

	active := &model.TenantEntity{Code: "t_active", Name: "active", Status: model.TenantStatusActive}
	suspended := &model.TenantEntity{Code: "t_suspended", Name: "suspended", Status: model.TenantStatusSuspended}
	for _, entity := range []*model.TenantEntity{active, suspended} {
		if err := db.Create(entity).Error; err != nil {
			t.Fatalf("insert tenant: %v", err)
		}
	}

	// tenantID 为空（无租户上下文的签发路径）：不校验
	if err := storage.tenantTokenGate(ctx, "", oidc.ErrInvalidGrant()); err != nil {
		t.Fatalf("empty tenantID should skip gate, got %v", err)
	}
	// active：放行
	if err := storage.tenantTokenGate(ctx, active.ID, oidc.ErrInvalidGrant()); err != nil {
		t.Fatalf("active tenant should pass gate, got %v", err)
	}
	// suspended + 授权码流：按调用方传入的 access_denied 返回
	codeFlowErr := storage.tenantTokenGate(ctx, suspended.ID, oidc.ErrAccessDenied())
	if codeFlowErr == nil {
		t.Fatal("suspended tenant should be rejected on authorization code flow")
	}
	if !strings.Contains(codeFlowErr.Error(), "tenant is suspended") {
		t.Errorf("code flow error should describe suspended tenant, got %v", codeFlowErr)
	}
	if !strings.Contains(codeFlowErr.Error(), "access_denied") {
		t.Errorf("code flow error should keep access_denied type, got %v", codeFlowErr)
	}
	// suspended + 刷新流：按调用方传入的 invalid_grant 返回
	refreshFlowErr := storage.tenantTokenGate(ctx, suspended.ID, oidc.ErrInvalidGrant())
	if refreshFlowErr == nil {
		t.Fatal("suspended tenant should be rejected on refresh flow")
	}
	if !strings.Contains(refreshFlowErr.Error(), "tenant is suspended") {
		t.Errorf("refresh flow error should describe suspended tenant, got %v", refreshFlowErr)
	}
	if !strings.Contains(refreshFlowErr.Error(), "invalid_grant") {
		t.Errorf("refresh flow error should keep invalid_grant type, got %v", refreshFlowErr)
	}
	// 租户行不存在：拒绝，但不能报成"已挂起"
	missingErr := storage.tenantTokenGate(ctx, "no-such-tenant-id", oidc.ErrInvalidGrant())
	if missingErr == nil {
		t.Fatal("missing tenant row should be rejected")
	}
	if !strings.Contains(missingErr.Error(), "tenant not found") {
		t.Errorf("missing tenant error should describe not found, got %v", missingErr)
	}
	if strings.Contains(missingErr.Error(), "tenant is suspended") {
		t.Errorf("missing tenant must not be reported as suspended, got %v", missingErr)
	}
}
