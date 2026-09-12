package svcapplicationclient

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// TestIsValidApplicationClientStatus 白名单本身：只认 enable/disable 与空值（空值语义为「不修改」）。
func TestIsValidApplicationClientStatus(t *testing.T) {
	for _, s := range []model.ApplicationClientStatus{"", model.ApplicationClientStatusEnable, model.ApplicationClientStatusDisable} {
		if !isValidApplicationClientStatus(s) {
			t.Fatalf("status=%q 应合法", s)
		}
	}
	for _, s := range []model.ApplicationClientStatus{"active", "inactive", "enabled", "ENABLE", "Enable", "1", "0", "on", " enable"} {
		if isValidApplicationClientStatus(s) {
			t.Fatalf("status=%q 应非法", s)
		}
	}
}

// TestApplicationClientUpdateRejectsIllegalStatus 校验白名单已接入 service 入口。
func TestApplicationClientUpdateRejectsIllegalStatus(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	entity := &model.ApplicationClientEntity{
		TenantID: "1",
		AppID:    "app1",
		Code:     "client1",
		Name:     "客户端1",
		Source:   model.ApplicationClientSourceThirdParty,
		Status:   model.ApplicationClientStatusEnable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewApplicationClientSvc()

	err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: entity.ID,
		Name:                "客户端1",
		Status:              "enabled",
	})
	if err == nil {
		t.Fatal("非法 status 应被拒绝")
	}
	if gerror.GetCode(err) != int(code.ApplicationClientUpdateError) {
		t.Fatalf("期望 ApplicationClientUpdateError, got %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.ApplicationClientStatusEnable {
		t.Fatalf("非法 status 不应落库, got %q", got.Status)
	}
}

// TestApplicationClientUpdateStatusEmptyMeansKeep 留空表示「不修改」，不得把状态覆盖为空串。
func TestApplicationClientUpdateStatusEmptyMeansKeep(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	entity := &model.ApplicationClientEntity{
		TenantID: "1",
		AppID:    "app1",
		Code:     "client2",
		Name:     "客户端2",
		Source:   model.ApplicationClientSourceThirdParty,
		Status:   model.ApplicationClientStatusEnable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewApplicationClientSvc()

	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: entity.ID, Name: "客户端2改名",
	}); err != nil {
		t.Fatalf("status 留空应允许: %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.ApplicationClientStatusEnable {
		t.Fatalf("留空不应覆盖状态, got %q", got.Status)
	}
	if got.Name != "客户端2改名" {
		t.Fatalf("其它字段应正常更新, got name=%q", got.Name)
	}
}
