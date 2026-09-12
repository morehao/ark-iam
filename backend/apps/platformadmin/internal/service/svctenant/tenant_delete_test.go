package svctenant

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// newTenantDeleteCtx 构造租户删除的请求上下文（删除链路不取 ctx.Request，仅需操作人/租户上下文）。
func newTenantDeleteCtx() *gin.Context {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, "other-tenant")
	ginCtx.Set(gcontext.KeyUserID, "0")
	return ginCtx
}

// TestTenantDeleteRejectsPlatformTenant 平台自运营租户（种子租户 t_platform）禁删：
// 它是平台控制台自身所在租户，删除即整栈失联、无恢复路径；判定按种子身份编码，
// 与 Update 的「不可挂起」同源（改名不影响判定）。
func TestTenantDeleteRejectsPlatformTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	platformTenant := &model.TenantEntity{
		Code: model.SeedPlatformTenantCode, Name: "平台运营中心",
		Type: model.TenantTypePlatform, Status: model.TenantStatusActive,
	}
	if err := db.Create(platformTenant).Error; err != nil {
		t.Fatalf("seed platform tenant: %v", err)
	}

	ctx := newTenantDeleteCtx()
	svc := NewTenantSvc()
	err := svc.Delete(ctx, &dtotenant.TenantDeleteReq{TenantID: platformTenant.ID})
	if err == nil {
		t.Fatal("删除平台租户必须被拒绝")
	}
	if gerror.GetCode(err) != int(code.TenantBuiltInDeleteForbiddenError) {
		t.Fatalf("期望 TenantBuiltInDeleteForbiddenError, got %v", err)
	}

	got, err := dao.NewTenantDao().GetByID(ctx, platformTenant.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatal("被拒删除的平台租户不得消失")
	}
}

// TestTenantDeleteAllowsCustomerTenant 客户租户不受内置禁删约束（删除人写入上下文用户）。
func TestTenantDeleteAllowsCustomerTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	customer := &model.TenantEntity{
		Code: "t_customer", Name: "客户租户",
		Type: model.TenantTypeCustomer, Status: model.TenantStatusActive,
	}
	if err := db.Create(customer).Error; err != nil {
		t.Fatalf("seed customer tenant: %v", err)
	}

	ctx := newTenantDeleteCtx()
	svc := NewTenantSvc()
	if err := svc.Delete(ctx, &dtotenant.TenantDeleteReq{TenantID: customer.ID}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	got, err := dao.NewTenantDao().GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil && got.ID != "" {
		t.Fatalf("expected tenant soft-deleted, got %+v", got)
	}

	var deleted model.TenantEntity
	if err := db.Unscoped().Where("id = ?", customer.ID).First(&deleted).Error; err != nil {
		t.Fatalf("query deleted row: %v", err)
	}
	if deleted.DeletedBy != "0" {
		t.Fatalf("expected deletedBy 0, got %s", deleted.DeletedBy)
	}
}
