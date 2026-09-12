package svctenantapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenantapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// TestIsValidTenantApplicationStatus 白名单本身：只认 enable/disable 与空值（空值语义为「不指定/不修改」）。
func TestIsValidTenantApplicationStatus(t *testing.T) {
	for _, s := range []model.TenantApplicationStatus{"", model.TenantApplicationStatusEnable, model.TenantApplicationStatusDisable} {
		if !isValidTenantApplicationStatus(s) {
			t.Fatalf("status=%q 应合法", s)
		}
	}
	for _, s := range []model.TenantApplicationStatus{"active", "inactive", "enabled", "ENABLE", "Enable", "1", "0", "on", " enable"} {
		if isValidTenantApplicationStatus(s) {
			t.Fatalf("status=%q 应非法", s)
		}
	}
}

// TestTenantApplicationCreateRejectsIllegalStatus 非法值在归属校验之前即被拦截。
func TestTenantApplicationCreateRejectsIllegalStatus(t *testing.T) {
	testutil.SetupSQLite(t, &model.TenantApplicationEntity{})

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewTenantApplicationSvc()

	_, err := svc.Create(ctx, &dtotenantapplication.TenantApplicationCreateReq{
		TenantID: "t1",
		AppID:    "app1",
		Status:   "enabled",
	})
	if err == nil {
		t.Fatal("非法 status 应被拒绝")
	}
	if gerror.GetCode(err) != int(code.TenantApplicationCreateError) {
		t.Fatalf("期望 TenantApplicationCreateError, got %v", err)
	}
}

// TestTenantApplicationUpdateRejectsIllegalStatus 校验白名单已接入更新入口。
func TestTenantApplicationUpdateRejectsIllegalStatus(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantApplicationEntity{})
	entity := &model.TenantApplicationEntity{
		TenantID:     "t1",
		AppID:        "app1",
		Status:       model.TenantApplicationStatusEnable,
		Config:       []byte("{}"),
		GrantedScope: []byte("[]"),
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewTenantApplicationSvc()

	err := svc.Update(ctx, &dtotenantapplication.TenantApplicationUpdateReq{
		TenantAppID: entity.ID,
		Status:      "enabled",
	})
	if err == nil {
		t.Fatal("非法 status 应被拒绝")
	}
	if gerror.GetCode(err) != int(code.TenantApplicationUpdateError) {
		t.Fatalf("期望 TenantApplicationUpdateError, got %v", err)
	}
	got, err := dao.NewTenantApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.TenantApplicationStatusEnable {
		t.Fatalf("非法 status 不应落库, got %q", got.Status)
	}
}
