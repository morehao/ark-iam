package svctenant

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objtenant"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// newTenantUpdateCtx 平台侧租户更新的上下文：租户上下文必须指向"别的租户"，
// 否则会先命中的是"不能挂起当前所在租户"的既有校验；Request 必须非空（挂起链路会取 ctx.Request.Context()）。
func newTenantUpdateCtx() *gin.Context {
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest(http.MethodPut, "/v1/platform/tenants/x", nil)
	ginCtx.Set(gcontext.KeyTenantID, "other-tenant")
	ginCtx.Set(gcontext.KeyUserID, "0")
	return ginCtx
}

// TestTenantUpdateRejectsPlatformTenantSuspend 平台自运营租户不可挂起：
// 它是平台控制台自身所在租户（挂起即整栈失联、无恢复路径），与种子的 status=reconcile
// 不变式同源；此处拒写以消除"控制台挂起、重启被种子纠正"的双写者。
func TestTenantUpdateRejectsPlatformTenantSuspend(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	platformTenant := &model.TenantEntity{
		Code: model.SeedPlatformTenantCode, Name: "平台运营中心",
		Type: model.TenantTypePlatform, Status: model.TenantStatusActive,
	}
	if err := db.Create(platformTenant).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := NewTenantSvc()
	ctx := newTenantUpdateCtx()

	err := svc.Update(ctx, &dtotenant.TenantUpdateReq{
		TenantID: platformTenant.ID,
		TenantBaseInfo: objtenant.TenantBaseInfo{
			Name: "平台运营中心", Type: model.TenantTypePlatform, Status: model.TenantStatusSuspended,
		},
	})
	if err == nil {
		t.Fatal("挂起平台租户必须被拒绝")
	}
	if gerror.GetCode(err) != int(code.TenantPlatformSuspendForbiddenError) {
		t.Fatalf("期望 TenantPlatformSuspendForbiddenError, got %v", err)
	}
	got, err := dao.NewTenantDao().GetByID(ctx, platformTenant.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.TenantStatusActive {
		t.Fatalf("被拒的挂起不得落库, got %q", got.Status)
	}

	// 平台租户改名归运维（create_only：L1 只在创建时写入），必须放行
	if err := svc.Update(ctx, &dtotenant.TenantUpdateReq{
		TenantID: platformTenant.ID,
		TenantBaseInfo: objtenant.TenantBaseInfo{
			Name: "ACME 平台运营中心", Type: model.TenantTypePlatform, Status: model.TenantStatusActive,
		},
	}); err != nil {
		t.Fatalf("平台租户改名应放行: %v", err)
	}
	got, err = dao.NewTenantDao().GetByID(ctx, platformTenant.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "ACME 平台运营中心" {
		t.Fatalf("平台租户改名未落库, got %q", got.Name)
	}

	// 客户租户挂起不受该规则约束
	customer := &model.TenantEntity{Code: "t_customer", Name: "客户租户", Type: model.TenantTypeCustomer, Status: model.TenantStatusActive}
	if err := db.Create(customer).Error; err != nil {
		t.Fatalf("seed customer tenant: %v", err)
	}
	if err := svc.Update(ctx, &dtotenant.TenantUpdateReq{
		TenantID: customer.ID,
		TenantBaseInfo: objtenant.TenantBaseInfo{
			Name: "客户租户", Type: model.TenantTypeCustomer, Status: model.TenantStatusSuspended,
		},
	}); err != nil {
		t.Fatalf("客户租户挂起应放行: %v", err)
	}
}
