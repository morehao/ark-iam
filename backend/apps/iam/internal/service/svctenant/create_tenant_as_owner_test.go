package svctenant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateTenantAsOwnerCreatesTenantUserAndSubscription(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = mustNewRequest(t)
	ginCtx.Set(gcontext.KeyUserID, uint(100))

	db := newCreateTenantAsOwnerTestDB(t)
	installTenantIamDB(t, db)
	seedApplication(t, db, 42, true)

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
	if users[0].CreatedBy != 88 {
		t.Fatalf("expected owner user createdBy to be personID 88, got %d", users[0].CreatedBy)
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

func TestCreateTenantAsOwnerWithoutAppRejected(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = mustNewRequest(t)
	ginCtx.Set(gcontext.KeyUserID, uint(101))

	db := newCreateTenantAsOwnerTestDB(t)
	installTenantIamDB(t, db)

	svc := &tenantSvc{}
	_, err := svc.CreateTenantAsOwner(ginCtx, &dtotenant.TenantCreateAsOwnerReq{
		PersonID: 89,
		Name:     "Beta",
		AppID:    0,
	})
	if err == nil {
		t.Fatalf("expected rejection when appID is 0, got nil error")
	}

	var tenants []model.TenantEntity
	if e := db.Find(&tenants).Error; e != nil {
		t.Fatalf("query tenants: %v", e)
	}
	if len(tenants) != 0 {
		t.Fatalf("expected no tenant created, got %d", len(tenants))
	}
}

func TestCreateTenantAsOwnerRejectedWhenAlreadyHasTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = mustNewRequest(t)
	ginCtx.Set(gcontext.KeyUserID, uint(102))

	db := newCreateTenantAsOwnerTestDB(t)
	installTenantIamDB(t, db)
	seedApplication(t, db, 43, true)

	now := time.Now()
	existingTenant := model.TenantEntity{Name: "Existing", Type: model.TenantTypeCustomer}
	if err := db.Create(&existingTenant).Error; err != nil {
		t.Fatalf("create existing tenant: %v", err)
	}
	if err := db.Create(&model.UserEntity{
		TenantID:   existingTenant.ID,
		PersonID:   90,
		Name:       "ExistingUser",
		Profile:    json.RawMessage("{}"),
		CustomData: json.RawMessage("{}"),
		IsOwner:    1,
		JoinedAt:   &now,
		CreatedBy:  90,
	}).Error; err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	svc := &tenantSvc{}
	_, err := svc.CreateTenantAsOwner(ginCtx, &dtotenant.TenantCreateAsOwnerReq{
		PersonID: 90,
		Name:     "Beta",
		AppID:    43,
	})
	if err == nil {
		t.Fatalf("expected rejection for person already having a tenant, got nil error")
	}
	targetErr, ok := err.(*gerror.Error)
	if !ok || targetErr.Code != code.TenantCreateAsOwnerForbiddenError {
		t.Fatalf("expected TenantCreateAsOwnerForbiddenError, got %v", err)
	}

	var tenants []model.TenantEntity
	if e := db.Where("id != ?", existingTenant.ID).Find(&tenants).Error; e != nil {
		t.Fatalf("query tenants: %v", e)
	}
	if len(tenants) != 0 {
		t.Fatalf("expected no new tenant created, got %d", len(tenants))
	}
}

func TestCreateTenantAsOwnerRejectedWhenPolicyForbids(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = mustNewRequest(t)
	ginCtx.Set(gcontext.KeyUserID, uint(103))

	db := newCreateTenantAsOwnerTestDB(t)
	installTenantIamDB(t, db)
	seedApplication(t, db, 44, false)

	svc := &tenantSvc{}
	_, err := svc.CreateTenantAsOwner(ginCtx, &dtotenant.TenantCreateAsOwnerReq{
		PersonID: 91,
		Name:     "Gamma",
		AppID:    44,
	})
	if err == nil {
		t.Fatalf("expected rejection when app policy forbids, got nil error")
	}
	targetErr, ok := err.(*gerror.Error)
	if !ok || targetErr.Code != code.TenantCreateAsOwnerForbiddenError {
		t.Fatalf("expected TenantCreateAsOwnerForbiddenError, got %v", err)
	}

	var tenants []model.TenantEntity
	if e := db.Find(&tenants).Error; e != nil {
		t.Fatalf("query tenants: %v", e)
	}
	if len(tenants) != 0 {
		t.Fatalf("expected no tenant created, got %d", len(tenants))
	}
}

func seedApplication(t *testing.T, db *gorm.DB, appID uint, allow bool) {
	t.Helper()
	policy := datatypes.JSON([]byte(`{"allowPersonCreateTenant":true}`))
	if !allow {
		policy = datatypes.JSON([]byte(`{"allowPersonCreateTenant":false}`))
	}
	app := model.ApplicationEntity{
		Model: gorm.Model{ID: appID},
		Name:  "TestApp",
		Type:  model.AppTypeFirstParty,
		Status: model.AppStatusEnable,
		TenantPolicy: policy,
	}
	if err := db.Create(&app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	if app.ID == 0 {
		t.Fatalf("seeded application has id 0")
	}
}

func newCreateTenantAsOwnerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeCreateTenantAsOwnerTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.TenantEntity{}, &model.UserEntity{}, &model.TenantApplicationEntity{}, &model.ApplicationEntity{}, &model.AuditLogEntity{}); err != nil {
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
