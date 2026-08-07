package svctenant

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateTenantAsOwnerCreatesTenantUserAndSubscription(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = mustNewRequest(t)
	ginCtx.Set(gcontext.KeyUserID, uint(100))

	db := newCreateTenantAsOwnerTestDB(t)
	installTenantIamDB(t, db)

	svc := &tenantSvc{}
	resp, err := svc.CreateTenantAsOwner(ginCtx, &dtotenant.TenantCreateAsOwnerReq{
		PersonID: 88,
		Name:     "Acme",
		AppID:    42,
	})
	if err != nil {
		t.Fatalf("CreateTenantAsOwner returned error: %v", err)
	}
	if resp.TenantID == 0 {
		t.Fatalf("expected non-zero tenant id")
	}

	var tenant model.TenantEntity
	if err := db.First(&tenant, resp.TenantID).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	if tenant.Name != "Acme" || tenant.CreatedBy != 100 {
		t.Fatalf("unexpected tenant: %+v", tenant)
	}

	var users []model.UserEntity
	if err := db.Where("tenant_id = ? AND person_id = ?", resp.TenantID, uint(88)).Find(&users).Error; err != nil {
		t.Fatalf("query users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 owner user, got %d", len(users))
	}
	if users[0].IsOwner != 1 || users[0].Name != "Acme" || users[0].TenantID != resp.TenantID || users[0].PersonID != 88 {
		t.Fatalf("unexpected owner user: %+v", users[0])
	}

	var apps []model.TenantApplicationEntity
	if err := db.Where("tenant_id = ? AND app_id = ?", resp.TenantID, uint(42)).Find(&apps).Error; err != nil {
		t.Fatalf("query tenant_application: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 tenant_application subscription, got %d", len(apps))
	}
	if apps[0].Status != "enable" {
		t.Fatalf("unexpected subscription status: %q", apps[0].Status)
	}
}

func TestCreateTenantAsOwnerWithoutAppSkipsSubscription(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = mustNewRequest(t)
	ginCtx.Set(gcontext.KeyUserID, uint(101))

	db := newCreateTenantAsOwnerTestDB(t)
	installTenantIamDB(t, db)

	svc := &tenantSvc{}
	resp, err := svc.CreateTenantAsOwner(ginCtx, &dtotenant.TenantCreateAsOwnerReq{
		PersonID: 89,
		Name:     "Beta",
		AppID:    0,
	})
	if err != nil {
		t.Fatalf("CreateTenantAsOwner returned error: %v", err)
	}
	if resp.TenantID == 0 {
		t.Fatalf("expected non-zero tenant id")
	}

	var apps []model.TenantApplicationEntity
	if err := db.Where("tenant_id = ?", resp.TenantID).Find(&apps).Error; err != nil {
		t.Fatalf("query tenant_application: %v", err)
	}
	if len(apps) != 0 {
		t.Fatalf("expected no tenant_application without appID, got %d", len(apps))
	}
}

func newCreateTenantAsOwnerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeCreateTenantAsOwnerTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.TenantEntity{}, &model.UserEntity{}, &model.TenantApplicationEntity{}, &model.AuditLogEntity{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func installTenantIamDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	prev := iamDBFromContext
	iamDBFromContext = func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}
	const svcName = "iam"
	dbclient.RegisterDBForTest(svcName, db)
	t.Cleanup(func() {
		iamDBFromContext = prev
		dbclient.ClearDBForTest(svcName)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}

func sanitizeCreateTenantAsOwnerTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}

func mustNewRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}
