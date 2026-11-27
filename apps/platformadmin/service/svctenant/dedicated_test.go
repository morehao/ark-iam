package svctenant

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDedicatedEnv 迁移 D onboarding 涉及的租户/域名/binding 及建租依赖表。
func setupDedicatedEnv(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:svctenant_d_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.TenantEntity{}, &model.DepartmentEntity{}, &model.PersonEntity{},
		&model.UserEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{},
		&model.MenuEntity{}, &model.RoleEntity{}, &model.RoleMenuEntity{},
		&model.TenantApplicationEntity{}, &model.UserRoleEntity{}, &model.UserApplicationEntity{},
		&model.RefreshTokenEntity{}, &model.DomainEntity{}, &model.ConsoleBindingEntity{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, _ := db.DB(); sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	seedTenantAdminApp(t, db)
	return db
}

func scopeCtx(tenantID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	gincontext.SetTenantScope(ctx, gcontext.CurrentScope(tenantID))
	ctx.Set(gcontext.KeyUserID, "platform-ops")
	return ctx
}

// TestDedicatedOnboardingCreatesPrimaryAndBinding：D 新建 → tenant provisioning + primary draft + binding draft。
func TestDedicatedOnboardingCreatesPrimaryAndBinding(t *testing.T) {
	db := setupDedicatedEnv(t)
	ctx := scopeCtx("")
	svc := &tenantSvc{db: dbclient.IamDB}

	req := newTenantCreateReq("Acme D", "admin@acme-d.com")
	req.IsolationMode = model.TenantIsolationDedicated
	req.PrimaryDomain = "login.acme.com"
	resp, err := svc.Create(requestCtx(ctx), req)
	if err != nil {
		t.Fatalf("Create D: %v", err)
	}

	tn, _ := accountGetTenant(t, db, resp.TenantID)
	if tn.Status != model.TenantStatusProvisioning {
		t.Fatalf("expected provisioning, got %s", tn.Status)
	}
	if tn.IsolationMode != model.TenantIsolationDedicated {
		t.Fatalf("expected dedicated, got %s", tn.IsolationMode)
	}

	var primary model.DomainEntity
	if err := db.WithContext(dbclient.ExplicitTenantContext(context.Background(), resp.TenantID)).
		Where("domain_role = ?", model.DomainRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("primary not created: %v", err)
	}
	if primary.Status != model.DomainStatusDraft || primary.Domain != "login.acme.com" {
		t.Fatalf("primary wrong: %+v", primary)
	}

	var binding model.ConsoleBindingEntity
	if err := db.WithContext(dbclient.CrossTenantContext(context.Background())).
		Where("tenant_id = ? AND kind = ?", resp.TenantID, model.ConsoleBindingKindTenant).First(&binding).Error; err != nil {
		t.Fatalf("binding not created: %v", err)
	}
	if binding.Status != model.ConsoleBindingStatusDraft || binding.Host != "login.acme.com" {
		t.Fatalf("binding wrong: %+v", binding)
	}
}

// TestDeleteDedicatedForbidden：D 租户 DELETE 一律 403。
func TestDeleteDedicatedForbidden(t *testing.T) {
	setupDedicatedEnv(t)
	ctx := scopeCtx("")
	svc := &tenantSvc{db: dbclient.IamDB}
	req := newTenantCreateReq("Acme D", "admin@acme-d.com")
	req.IsolationMode = model.TenantIsolationDedicated
	req.PrimaryDomain = "login.acme.com"
	resp, err := svc.Create(requestCtx(ctx), req)
	if err != nil {
		t.Fatalf("Create D: %v", err)
	}
	err = svc.Delete(requestCtx(ctx), &dtotenant.TenantDeleteReq{TenantID: resp.TenantID, Operator: operatorOf(ctx)})
	if err == nil {
		t.Fatal("expected DELETE of dedicated tenant to be forbidden")
	}
}

// TestOffboardSuspendsTenantAndDisablesPrimary：offboard → tenant suspended、primary disabled、issuer 清空。
func TestOffboardSuspendsTenantAndDisablesPrimary(t *testing.T) {
	db := setupDedicatedEnv(t)
	ctx := scopeCtx("")
	svc := &tenantSvc{db: dbclient.IamDB}
	req := newTenantCreateReq("Acme D", "admin@acme-d.com")
	req.IsolationMode = model.TenantIsolationDedicated
	req.PrimaryDomain = "login.acme.com"
	resp, err := svc.Create(requestCtx(ctx), req)
	if err != nil {
		t.Fatalf("Create D: %v", err)
	}
	tid := resp.TenantID

	// 手动把 tenant 置 active + primary active + issuer（模拟 onboarding 完成）。
	setTenantActive(t, db, tid, "https://login.acme.com/oidc")
	db.WithContext(dbclient.ExplicitTenantContext(context.Background(), tid)).
		Model(&model.DomainEntity{}).Where("domain_role = ?", model.DomainRolePrimary).Update("status", model.DomainStatusActive)

	if err := svc.Offboard(requestCtx(ctx), &dtotenant.TenantOffboardReq{TenantID: tid, Operator: operatorOf(ctx)}); err != nil {
		t.Fatalf("Offboard: %v", err)
	}
	tn, _ := accountGetTenant(t, db, tid)
	if tn.Status != model.TenantStatusSuspended {
		t.Fatalf("expected suspended, got %s", tn.Status)
	}
	if tn.Issuer != "" {
		t.Fatalf("expected issuer cleared, got %q", tn.Issuer)
	}
	var primary model.DomainEntity
	db.WithContext(dbclient.ExplicitTenantContext(context.Background(), tid)).
		Where("domain_role = ?", model.DomainRolePrimary).First(&primary)
	if primary.Status != model.DomainStatusDisabled {
		t.Fatalf("expected primary disabled, got %s", primary.Status)
	}
}

func accountGetTenant(t *testing.T, db *gorm.DB, id string) (*model.TenantEntity, error) {
	t.Helper()
	var tn model.TenantEntity
	err := db.WithContext(dbclient.CrossTenantContext(context.Background())).Where("id = ?", id).First(&tn).Error
	return &tn, err
}

func setTenantActive(t *testing.T, db *gorm.DB, id, issuer string) {
	t.Helper()
	if err := db.WithContext(dbclient.CrossTenantContext(context.Background())).Model(&model.TenantEntity{}).
		Where("id = ?", id).Updates(map[string]any{"status": model.TenantStatusActive, "issuer": issuer}).Error; err != nil {
		t.Fatalf("set tenant active: %v", err)
	}
}
