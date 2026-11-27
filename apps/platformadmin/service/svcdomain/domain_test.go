package svcdomain

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtodomain"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB 内存 SQLite 注册 iam 库，迁移租户/域名表。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:svcdomain_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.TenantEntity{}, &model.DomainEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, _ := db.DB(); sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newGinCtx(tenantID, userID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	if ctx.Request == nil {
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	}
	ctx.Set(gcontext.KeyTenantID, tenantID)
	gincontext.SetTenantScope(ctx, gcontext.CurrentScope(tenantID))
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

// seedTenant 插入租户（iam_tenant 是 skip 表，CrossTenant 写）。
func seedTenant(t *testing.T, db *gorm.DB, id string, status model.TenantStatus) {
	t.Helper()
	tn := &model.TenantEntity{Status: status, IsolationMode: model.TenantIsolationDedicated}
	tn.ID = id
	if err := db.WithContext(dbclient.CrossTenantContext(context.Background())).Create(tn).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
}

func getDomain(t *testing.T, db *gorm.DB, tenantID, id string) *model.DomainEntity {
	t.Helper()
	var d model.DomainEntity
	if err := db.WithContext(dbclient.ExplicitTenantContext(context.Background(), tenantID)).Where("id=?", id).First(&d).Error; err != nil {
		t.Fatalf("get domain: %v", err)
	}
	return &d
}

// TestDomainLifecycle 覆盖 primary draft→verified→tls_ready→active，tenant 被激活。
func TestDomainLifecycle(t *testing.T) {
	db := newTestDB(t)
	seedTenant(t, db, "t1", model.TenantStatusProvisioning)
	ctx := newGinCtx("t1", "u1")
	reqCtx := requestCtx(ctx)
	svc := NewDomainSvc(nil)

	cr, err := svc.Create(reqCtx, &dtodomain.DomainCreateReq{
		TenantID: "t1", Domain: "login.acme.com",
		DomainRole: model.DomainRolePrimary, DomainUsage: model.DomainUsageLogin,
		Operator: operatorOf(ctx),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if getDomain(t, db, "t1", cr.ID).Status != model.DomainStatusDraft {
		t.Fatal("expected draft after create")
	}
	if _, err := svc.Verify(reqCtx, &dtodomain.DomainVerifyReq{DomainID: cr.ID, Operator: operatorOf(ctx)}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if _, err := svc.TLSReady(reqCtx, &dtodomain.DomainTLSReadyReq{DomainID: cr.ID, Operator: operatorOf(ctx)}); err != nil {
		t.Fatalf("TLSReady: %v", err)
	}
	if _, err := svc.Activate(reqCtx, &dtodomain.DomainActivateReq{DomainID: cr.ID, Operator: operatorOf(ctx)}); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	d := getDomain(t, db, "t1", cr.ID)
	if d.Status != model.DomainStatusActive {
		t.Fatalf("expected active, got %s", d.Status)
	}
	// tenant 被激活 + issuer 派生
	var tn model.TenantEntity
	db.WithContext(dbclient.CrossTenantContext(context.Background())).Where("id=?", "t1").First(&tn)
	if tn.Status != model.TenantStatusActive {
		t.Fatalf("expected tenant active, got %s", tn.Status)
	}
	if tn.Issuer != "https://login.acme.com/oidc" {
		t.Fatalf("issuer = %q", tn.Issuer)
	}
}

// TestAliasRequiresTarget 覆盖 alias 无 target 被拒。
func TestAliasRequiresTarget(t *testing.T) {
	db := newTestDB(t)
	seedTenant(t, db, "t1", model.TenantStatusActive)
	ctx := newGinCtx("t1", "u1")
	svc := NewDomainSvc(nil)
	_, err := svc.Create(requestCtx(ctx), &dtodomain.DomainCreateReq{
		TenantID: "t1", Domain: "sso.acme.com",
		DomainRole: model.DomainRoleAlias, DomainUsage: model.DomainUsageLogin,
		Operator: operatorOf(ctx),
	})
	if err == nil {
		t.Fatal("expected error for alias without target")
	}
}

// TestPrimaryEmailUsageRejected 覆盖 primary usage=email 被拒。
func TestPrimaryEmailUsageRejected(t *testing.T) {
	db := newTestDB(t)
	seedTenant(t, db, "t1", model.TenantStatusActive)
	ctx := newGinCtx("t1", "u1")
	svc := NewDomainSvc(nil)
	_, err := svc.Create(requestCtx(ctx), &dtodomain.DomainCreateReq{
		TenantID: "t1", Domain: "acme.com",
		DomainRole: model.DomainRolePrimary, DomainUsage: model.DomainUsageEmail,
		Operator: operatorOf(ctx),
	})
	if err == nil {
		t.Fatal("expected error for primary with usage=email")
	}
}

// TestDeleteRejectedWhenReferencedByAlias 覆盖有未 released alias 引用时删除被拒。
func TestDeleteRejectedWhenReferencedByAlias(t *testing.T) {
	db := newTestDB(t)
	seedTenant(t, db, "t1", model.TenantStatusProvisioning)
	ctx := newGinCtx("t1", "u1")
	svc := NewDomainSvc(nil)
	reqCtx := requestCtx(ctx)

	pRes, err := svc.Create(reqCtx, &dtodomain.DomainCreateReq{
		TenantID: "t1", Domain: "login.acme.com",
		DomainRole: model.DomainRolePrimary, DomainUsage: model.DomainUsageLogin, Operator: operatorOf(ctx),
	})
	if err != nil {
		t.Fatalf("create primary: %v", err)
	}
	if _, err := svc.Create(reqCtx, &dtodomain.DomainCreateReq{
		TenantID: "t1", Domain: "sso.acme.com",
		DomainRole: model.DomainRoleAlias, DomainUsage: model.DomainUsageLogin, TargetDomain: "login.acme.com",
		Operator: operatorOf(ctx),
	}); err != nil {
		t.Fatalf("create alias: %v", err)
	}
	if err := svc.Delete(reqCtx, &dtodomain.DomainDeleteReq{DomainID: pRes.ID, Operator: operatorOf(ctx)}); err == nil {
		t.Fatal("expected delete rejected when referenced by alias")
	}
}

var _ = dao.NewDomainDao
