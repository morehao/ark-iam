package svcapplication

import (
	"github.com/morehao/ark-iam/pkg/dbclient"
	"testing"

	"gorm.io/gorm"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoapplication"
	"github.com/morehao/ark-iam/apps/platformadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gerror"
)

// seedOwnerTenant 落一条归属租户（iam_tenant 是全局表，控制台建应用时 owner_type=tenant 需其存在）。
func seedOwnerTenant(t *testing.T, db *gorm.DB, tenantID string) {
	t.Helper()
	if err := db.Create(&model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: tenantID}},
		Code:       "code-" + tenantID,
		Name:       "归属租户-" + tenantID,
		Status:     model.TenantStatusActive,
	}).Error; err != nil {
		t.Fatalf("seed owner tenant %s: %v", tenantID, err)
	}
}

// TestApplicationCreateRejectsInvalidCode 应用编码规则为下划线连接（model.AppCodePattern：
// 小写字母开头，仅含小写字母/数字/下划线）：连字符、大写、数字开头等非法编码必须在
// service 入口被拒（返回 ApplicationCodeInvalidError），不得落库。
func TestApplicationCreateRejectsInvalidCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.TenantEntity{})
	seedOwnerTenant(t, db, "t_owner")

	svc := NewApplicationSvc(nil)
	for _, badCode := range []string{"my-app", "My_App", "1app", "app.web", "app web", "_app", ""} {
		_, err := svc.Create(requestCtx(newApplicationCtx()), &dtoapplication.ApplicationCreateReq{Code: badCode, Name: "非法编码应用", OwnerTenantID: "t_owner", Operator: operatorOf(newApplicationCtx())})
		if err == nil {
			t.Fatalf("Create(%q) expected error", badCode)
		}
		if gerror.GetCode(err) != int(code.ApplicationCodeInvalidError) {
			t.Fatalf("Create(%q) error code = %d, want %d", badCode, gerror.GetCode(err), code.ApplicationCodeInvalidError)
		}
	}
}

// TestApplicationCreateAcceptsUnderscoreCode 合法编码（下划线形态）落库成功，
// 且控制台创建的应用恒为 third_party、归属固定 owner_type=tenant（租户 id 由入参给出）。
func TestApplicationCreateAcceptsUnderscoreCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.TenantEntity{})
	seedOwnerTenant(t, db, "t_owner")

	ctx := newApplicationCtx()
	resp, err := NewApplicationSvc(nil).Create(requestCtx(ctx), &dtoapplication.ApplicationCreateReq{Code: "my_app_2", Name: "合法应用", OwnerTenantID: "t_owner", Operator: operatorOf(ctx)})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Code != "my_app_2" {
		t.Fatalf("resp code = %q, want my_app_2", resp.Code)
	}

	got, err := dao.NewApplicationDao().GetByID(dbclient.CrossTenantContext(ctx), resp.AppID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatalf("application not persisted: %+v", got)
	}
	if got.Code != "my_app_2" {
		t.Errorf("stored code = %q, want my_app_2", got.Code)
	}
	if got.Source != model.AppSourceThirdParty {
		t.Errorf("stored source = %q, want %q", got.Source, model.AppSourceThirdParty)
	}
	// 控制台建第三方应用固定 owner_type=tenant，owner_tenant_id=入参
	if got.OwnerType != model.AppOwnerTypeTenant {
		t.Errorf("stored ownerType = %q, want %q", got.OwnerType, model.AppOwnerTypeTenant)
	}
	if got.OwnerTenantID != "t_owner" {
		t.Errorf("stored ownerTenantID = %q, want t_owner", got.OwnerTenantID)
	}
}

// TestApplicationCreateRejectsInvalidOwner 归属入参非法必须被拒（ApplicationOwnerInvalidError）：
// owner_type 取值不在白名单、或 tenant 但 owner_tenant_id 为空。
func TestApplicationCreateRejectsInvalidOwner(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.TenantEntity{})
	seedOwnerTenant(t, db, "t_owner")

	svc := NewApplicationSvc(nil)
	cases := []struct {
		name      string
		ownerType model.AppOwnerType
		tenantID  string
	}{
		{"非法 owner_type 取值", model.AppOwnerType("galaxy"), ""},
		{"owner_type=tenant 但 tenant id 为空", model.AppOwnerTypeTenant, ""},
	}
	for _, tc := range cases {
		ctx := newApplicationCtx()
		_, err := svc.Create(requestCtx(ctx), &dtoapplication.ApplicationCreateReq{
			Code: "owner_app", Name: "归属非法应用",
			OwnerType: tc.ownerType, OwnerTenantID: tc.tenantID,
			Operator: operatorOf(ctx),
		})
		if err == nil || gerror.GetCode(err) != int(code.ApplicationOwnerInvalidError) {
			t.Fatalf("%s: expected ApplicationOwnerInvalidError, got %v", tc.name, err)
		}
	}
}

// TestApplicationCreateRejectsNonExistentOwnerTenant 归属租户不存在时拒绝（与系统错误分离）。
func TestApplicationCreateRejectsNonExistentOwnerTenant(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.TenantEntity{})

	ctx := newApplicationCtx()
	_, err := NewApplicationSvc(nil).Create(requestCtx(ctx), &dtoapplication.ApplicationCreateReq{
		Code: "ghost_app", Name: "悬空归属应用",
		OwnerType: model.AppOwnerTypeTenant, OwnerTenantID: "no_such_tenant",
		Operator: operatorOf(ctx),
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationOwnerInvalidError) {
		t.Fatalf("expected ApplicationOwnerInvalidError for non-existent tenant, got %v", err)
	}
}
