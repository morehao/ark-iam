package svcauth

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// TestIsValidConnectorStatus 白名单本身：只认 enable/disable 与空值（空值语义为「不指定/不修改」）。
// 取值统一见 status-source-consistency-design-20260912.md D1/D4。
func TestIsValidConnectorStatus(t *testing.T) {
	for _, s := range []model.ConnectorStatus{"", model.ConnectorStatusEnable, model.ConnectorStatusDisable} {
		if !isValidConnectorStatus(s) {
			t.Fatalf("status=%q 应合法", s)
		}
	}
	// 历史值 enabled、同类拼写与大小写、数字等脏值一律拒绝
	for _, s := range []model.ConnectorStatus{"enabled", "disabled", "active", "inactive", "ENABLE", "Enable", "1", "0", "on", " enable"} {
		if isValidConnectorStatus(s) {
			t.Fatalf("status=%q 应非法", s)
		}
	}
}

// TestConnectorCreateRejectsIllegalStatus 校验白名单已接入创建入口，历史值 "enabled" 已被拒绝。
func TestConnectorCreateRejectsIllegalStatus(t *testing.T) {
	testutil.SetupSQLite(t, &model.ConnectorEntity{})

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewConnectorSvc()

	_, err := svc.Create(ctx, &dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			Name:   "连接器1",
			Status: "enabled", // 阶段 2 前的历史取值，已不在白名单内
		},
	})
	if err == nil {
		t.Fatal("非法 status 应被拒绝")
	}
	if gerror.GetCode(err) != int(code.ConnectorCreateError) {
		t.Fatalf("期望 ConnectorCreateError, got %v", err)
	}
}

// TestConnectorCreateWithoutStatusDefaultsEnable 未指定 status 时落默认值 enable（D4）：
// 历史行为是落空串 ”，而鉴权判 `!= enable` 会直接拒绝，导致连接器创建后永久不可用。
func TestConnectorCreateWithoutStatusDefaultsEnable(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewConnectorSvc()

	resp, err := svc.Create(ctx, &dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			Name:         "连接器1",
			Protocol:     "oauth2",
			Provider:     "github",
			ClaimMapping: map[string]any{"email": "email"},
			DomainPolicy: map[string]any{"mode": "allow_all"},
		},
	})
	if err != nil {
		t.Fatalf("未指定 status 应允许创建: %v", err)
	}

	var entity model.ConnectorEntity
	if err := db.Where("id = ?", resp.ConnectorID).First(&entity).Error; err != nil {
		t.Fatalf("查询连接器失败: %v", err)
	}
	if entity.Status != model.ConnectorStatusEnable {
		t.Fatalf("未指定 status 应落默认值 enable, got %q", entity.Status)
	}
}

// TestConnectorUpdateRejectsIllegalStatus 校验白名单已接入更新入口，且非法值不落库。
func TestConnectorUpdateRejectsIllegalStatus(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})
	entity := &model.ConnectorEntity{
		TenantID:     "1",
		Name:         "连接器1",
		Protocol:     "oauth2",
		Provider:     "github",
		Status:       model.ConnectorStatusEnable,
		Config:       []byte("{}"),
		ClaimMapping: []byte("{}"),
		DomainPolicy: []byte("{}"),
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewConnectorSvc()

	err := svc.Update(ctx, &dtoauth.ConnectorUpdateReq{
		ConnectorID:       entity.ID,
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{Name: "连接器1", Status: "enabled"},
	})
	if err == nil {
		t.Fatal("非法 status 应被拒绝")
	}
	if gerror.GetCode(err) != int(code.ConnectorUpdateError) {
		t.Fatalf("期望 ConnectorUpdateError, got %v", err)
	}
	got, err := dao.NewConnectorDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.ConnectorStatusEnable {
		t.Fatalf("非法 status 不应落库, got %q", got.Status)
	}
}

// TestConnectorUpdateStatusDisable 合法值 disable 可正常流转（阶段 2 新增的另一半值域）。
func TestConnectorUpdateStatusDisable(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})
	entity := &model.ConnectorEntity{
		TenantID:     "1",
		Name:         "连接器1",
		Protocol:     "oauth2",
		Provider:     "github",
		Status:       model.ConnectorStatusEnable,
		Config:       []byte("{}"),
		ClaimMapping: []byte("{}"),
		DomainPolicy: []byte("{}"),
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewConnectorSvc()

	if err := svc.Update(ctx, &dtoauth.ConnectorUpdateReq{
		ConnectorID:       entity.ID,
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{Name: "连接器1", Status: model.ConnectorStatusDisable},
	}); err != nil {
		t.Fatalf("合法 status disable 应允许: %v", err)
	}
	got, err := dao.NewConnectorDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.ConnectorStatusDisable {
		t.Fatalf("status 应更新为 disable, got %q", got.Status)
	}
}
