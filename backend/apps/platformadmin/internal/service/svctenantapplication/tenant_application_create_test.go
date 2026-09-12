package svctenantapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenantapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// newCreateCtx 构造请求上下文：ctx 租户固定为 t1，用于验证平台侧按请求参数跨租户运维
// ——订阅归属取自 req.TenantID，而非调用者 token 里的租户。
func newCreateCtx() *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "t1")
	ctx.Set(gcontext.KeyUserID, "u1")
	return ctx
}

func seedTenantEntity(t *testing.T, db *gorm.DB, tenantID, name string) {
	t.Helper()
	if err := db.Create(&model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: tenantID}},
		Code:       "code-" + tenantID,
		Name:       name,
		Status:     model.TenantStatusActive,
	}).Error; err != nil {
		t.Fatalf("seed tenant %s: %v", tenantID, err)
	}
}

func seedAppEntity(t *testing.T, db *gorm.DB, appID, name string) {
	t.Helper()
	if err := db.Create(&model.ApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: appID}},
		Code:       "code-" + appID,
		Name:       name,
		Status:     model.AppStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed application %s: %v", appID, err)
	}
}

func setupTenantAppDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.SetupSQLite(t,
		&model.TenantEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
}

// TestCreateRejectsUnknownTenant 归属租户必须存在：订阅不能指向不存在的租户。
func TestCreateRejectsUnknownTenant(t *testing.T) {
	setupTenantAppDB(t)

	_, err := NewTenantApplicationSvc().Create(newCreateCtx(), &dtotenantapplication.TenantApplicationCreateReq{
		TenantID: "no-such-tenant",
		AppID:    "app-1",
	})
	if err != code.GetError(code.TenantNotExistError) {
		t.Fatalf("expected TenantNotExistError, got %v", err)
	}
}

// TestCreateRejectsUnknownApp 订阅必须指向真实存在的应用：
// 悬空 appID 会让租户侧按订阅反查应用时静默丢弃（loadSubscribedApps），留下永不生效的订阅。
func TestCreateRejectsUnknownApp(t *testing.T) {
	db := setupTenantAppDB(t)
	seedTenantEntity(t, db, "t2", "租户二")

	_, err := NewTenantApplicationSvc().Create(newCreateCtx(), &dtotenantapplication.TenantApplicationCreateReq{
		TenantID: "t2",
		AppID:    "no-such-app",
	})
	if err != code.GetError(code.ApplicationNotExistError) {
		t.Fatalf("expected ApplicationNotExistError, got %v", err)
	}
}

// TestCreateRejectsDuplicateSubscription 同一租户对同一应用只允许一条订阅：
// tenant_application 无唯一索引，重复必须由应用层拦截。
func TestCreateRejectsDuplicateSubscription(t *testing.T) {
	db := setupTenantAppDB(t)
	seedTenantEntity(t, db, "t2", "租户二")
	seedAppEntity(t, db, "app-1", "应用一")

	ctx := newCreateCtx()
	svc := NewTenantApplicationSvc()
	req := &dtotenantapplication.TenantApplicationCreateReq{TenantID: "t2", AppID: "app-1"}
	if _, err := svc.Create(ctx, req); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if _, err := svc.Create(ctx, req); err != code.GetError(code.TenantApplicationExistError) {
		t.Fatalf("expected TenantApplicationExistError, got %v", err)
	}
}

// TestCreateTargetsRequestTenantAndPageListFilters 平台侧可给任意租户开通订阅
// （ctx 租户是 t1，实际开通到 t2/t3）；列表按 tenantID 筛选，并回填租户名/应用名。
func TestCreateTargetsRequestTenantAndPageListFilters(t *testing.T) {
	db := setupTenantAppDB(t)
	seedTenantEntity(t, db, "t2", "租户二")
	seedTenantEntity(t, db, "t3", "租户三")
	seedAppEntity(t, db, "app-1", "应用一")

	ctx := newCreateCtx()
	svc := NewTenantApplicationSvc()
	for _, tenantID := range []string{"t2", "t3"} {
		if _, err := svc.Create(ctx, &dtotenantapplication.TenantApplicationCreateReq{TenantID: tenantID, AppID: "app-1"}); err != nil {
			t.Fatalf("create for %s failed: %v", tenantID, err)
		}
	}

	// 不传 tenantID：跨租户全量视角（平台侧）
	all, err := svc.PageList(ctx, &dtotenantapplication.TenantApplicationPageListReq{})
	if err != nil {
		t.Fatalf("page list all: %v", err)
	}
	if len(all.List) != 2 {
		t.Fatalf("expected 2 subscriptions across tenants, got %+v", all.List)
	}

	// 传 tenantID：只返回该租户的订阅，并带出名称
	one, err := svc.PageList(ctx, &dtotenantapplication.TenantApplicationPageListReq{TenantID: "t2"})
	if err != nil {
		t.Fatalf("page list t2: %v", err)
	}
	if len(one.List) != 1 {
		t.Fatalf("expected 1 subscription for t2, got %+v", one.List)
	}
	item := one.List[0]
	if item.TenantID != "t2" || item.TenantName != "租户二" {
		t.Fatalf("unexpected tenant fields: %+v", item)
	}
	if item.AppID != "app-1" || item.AppName != "应用一" {
		t.Fatalf("unexpected app fields: %+v", item)
	}
}

// TestDeleteOtherTenantSubscription 平台侧可删除任意租户的订阅（与 /v1/platform/tenants 同一信任模型，不做 ctx 租户归属校验）。
func TestDeleteOtherTenantSubscription(t *testing.T) {
	db := setupTenantAppDB(t)
	seedTenantEntity(t, db, "t2", "租户二")
	seedAppEntity(t, db, "app-1", "应用一")

	ctx := newCreateCtx()
	svc := NewTenantApplicationSvc()
	created, err := svc.Create(ctx, &dtotenantapplication.TenantApplicationCreateReq{TenantID: "t2", AppID: "app-1"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.Delete(ctx, &dtotenantapplication.TenantApplicationDeleteReq{TenantAppID: created.TenantAppID}); err != nil {
		t.Fatalf("delete other tenant subscription failed: %v", err)
	}
	if err := svc.Delete(ctx, &dtotenantapplication.TenantApplicationDeleteReq{TenantAppID: created.TenantAppID}); err != code.GetError(code.TenantApplicationNotExistError) {
		t.Fatalf("expected TenantApplicationNotExistError on second delete, got %v", err)
	}
}
