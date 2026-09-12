package svcapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// TestIsValidAppStatus 白名单本身：只认 enable/disable 与空值（空值语义为「不修改」）。
func TestIsValidAppStatus(t *testing.T) {
	valid := []model.AppStatus{"", model.AppStatusEnable, model.AppStatusDisable}
	for _, s := range valid {
		if !isValidAppStatus(s) {
			t.Fatalf("status=%q 应合法", s)
		}
	}
	// 非法值：含历史/同类拼写（active/enabled）与大小写、数字等常见脏值
	for _, s := range []model.AppStatus{"active", "inactive", "enabled", "disabled", "ENABLE", "Enable", "1", "0", "on", " enable"} {
		if isValidAppStatus(s) {
			t.Fatalf("status=%q 应非法", s)
		}
	}
}

// TestApplicationUpdateRejectsIllegalStatus 校验白名单已接入 service 入口：
// 非法值返回功能级错误码，且不得落库。
func TestApplicationUpdateRejectsIllegalStatus(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	entity := &model.ApplicationEntity{
		Code:   "app1",
		Name:   "应用1",
		Source: model.AppSourceThirdParty,
		Status: model.AppStatusEnable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewApplicationSvc()

	err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:  entity.ID,
		Name:   "应用1",
		Status: "enabled",
	})
	if err == nil {
		t.Fatal("非法 status 应被拒绝")
	}
	if gerror.GetCode(err) != int(code.ApplicationUpdateError) {
		t.Fatalf("期望 ApplicationUpdateError, got %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.AppStatusEnable {
		t.Fatalf("非法 status 不应落库, got %q", got.Status)
	}
}

// TestApplicationUpdateStatusEmptyMeansKeep 留空表示「不修改」：
// 不得把已有状态覆盖为空串（历史行为会把 status 写成 ”，使应用既非启用也非停用）。
func TestApplicationUpdateStatusEmptyMeansKeep(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	entity := &model.ApplicationEntity{
		Code:   "app2",
		Name:   "应用2",
		Source: model.AppSourceThirdParty,
		Status: model.AppStatusEnable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewApplicationSvc()

	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{AppID: entity.ID, Name: "应用2改名"}); err != nil {
		t.Fatalf("status 留空应允许: %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.AppStatusEnable {
		t.Fatalf("留空不应覆盖状态, got %q", got.Status)
	}
	if got.Name != "应用2改名" {
		t.Fatalf("其它字段应正常更新, got name=%q", got.Name)
	}

	// 合法值仍可正常流转
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID: entity.ID, Name: "应用2改名", Status: model.AppStatusDisable,
	}); err != nil {
		t.Fatalf("合法 status 应允许: %v", err)
	}
	got, err = dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.AppStatusDisable {
		t.Fatalf("合法 status 应生效, got %q", got.Status)
	}
}
