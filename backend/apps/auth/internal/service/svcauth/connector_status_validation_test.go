package svcauth

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// TestIsValidConnectorStatus 白名单本身：只认 enable/disable 与空值（空值语义为「不指定/不修改」）。
// 取值统一见 docs/design/glossary.md「启停状态」。
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
			ClaimMapping: model.ConnectorClaimMapping{Email: "email"},
			DomainPolicy: model.ConnectorDomainPolicy{AllowedDomains: model.DomainList{"example.com"}},
		},
	})
	if err != nil {
		t.Fatalf("未指定 status 应允许创建: %v", err)
	}

	var entity model.ConnectorEntity
	// 断言使用与 svc.Create 同一个租户上下文，避免绕过租户隔离插件
	if err := db.WithContext(ctx).Where("id = ?", resp.ConnectorID).First(&entity).Error; err != nil {
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
		Config:       model.ConnectorConfig{},
		ClaimMapping: model.ConnectorClaimMapping{},
		DomainPolicy: model.ConnectorDomainPolicy{},
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
		Config:       model.ConnectorConfig{},
		ClaimMapping: model.ConnectorClaimMapping{},
		DomainPolicy: model.ConnectorDomainPolicy{},
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

// TestNormalizeConnectorSwitch 开关枚举白名单本身：空串归一为 disable（与列默认值一致），
// enable/disable 原样通过，布尔字面量/数字/大小写脏值一律拒绝。
func TestNormalizeConnectorSwitch(t *testing.T) {
	enable, disable := model.ConnectorAutoCreateUserFlagEnable, model.ConnectorAutoCreateUserFlagDisable
	cases := []struct {
		in     model.ConnectorAutoCreateUserFlag
		want   model.ConnectorAutoCreateUserFlag
		wantOK bool
	}{
		{in: "", want: disable, wantOK: true},
		{in: enable, want: enable, wantOK: true},
		{in: disable, want: disable, wantOK: true},
		{in: "true", want: "", wantOK: false},
		{in: "false", want: "", wantOK: false},
		{in: "1", want: "", wantOK: false},
		{in: "0", want: "", wantOK: false},
		{in: "ENABLE", want: "", wantOK: false},
		{in: " enable", want: "", wantOK: false},
	}
	for _, tc := range cases {
		got, ok := normalizeConnectorSwitch(tc.in, enable, disable)
		if ok != tc.wantOK || got != tc.want {
			t.Fatalf("normalizeConnectorSwitch(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

// TestConnectorCreateRejectsIllegalSwitch 四个开关列同属前端枚举入参，非法值必须在
// service 入口被拒（且不落库），错误码沿用该操作既有的 ConnectorCreateError。
func TestConnectorCreateRejectsIllegalSwitch(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewConnectorSvc()

	_, err := svc.Create(ctx, &dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			Name:                "连接器1",
			Protocol:            "oauth2",
			Provider:            "github",
			Status:              model.ConnectorStatusEnable,
			AllowAutoCreateUser: "true", // 阶段 3 前的布尔字面量，已不在枚举白名单内
		},
	})
	if err == nil {
		t.Fatal("非法开关值应被拒绝")
	}
	if gerror.GetCode(err) != int(code.ConnectorCreateError) {
		t.Fatalf("期望 ConnectorCreateError, got %v", err)
	}
	var count int64
	if err := db.WithContext(ctx).Model(&model.ConnectorEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("非法开关值不应落库, got %d rows", count)
	}
}

// TestConnectorCreateWithoutSwitchesDefaultsDisable 未提交开关字段时落 disable（列默认值），
// 保持改造前"未提交 bool 即 false"的既有行为。
func TestConnectorCreateWithoutSwitchesDefaultsDisable(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewConnectorSvc()

	resp, err := svc.Create(ctx, &dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			Name:     "连接器1",
			Protocol: "oauth2",
			Provider: "github",
		},
	})
	if err != nil {
		t.Fatalf("未提交开关字段应允许创建: %v", err)
	}

	var entity model.ConnectorEntity
	if err := db.WithContext(ctx).Where("id = ?", resp.ConnectorID).First(&entity).Error; err != nil {
		t.Fatalf("查询连接器失败: %v", err)
	}
	if entity.AllowAutoCreateUser != model.ConnectorAutoCreateUserFlagDisable ||
		entity.AllowAccountLink != model.ConnectorAccountLinkFlagDisable ||
		entity.SyncProfile != model.ConnectorSyncProfileFlagDisable ||
		entity.EnableTokenStorage != model.ConnectorTokenStorageFlagDisable {
		t.Fatalf("未提交的开关应落 disable, got %q/%q/%q/%q",
			entity.AllowAutoCreateUser, entity.AllowAccountLink, entity.SyncProfile, entity.EnableTokenStorage)
	}
}

// TestConnectorUpdateRejectsIllegalSwitch 更新入口同样做白名单校验：非法开关值不落库。
func TestConnectorUpdateRejectsIllegalSwitch(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})
	entity := &model.ConnectorEntity{
		TenantID:            "1",
		Name:                "连接器1",
		Protocol:            "oauth2",
		Provider:            "github",
		Status:              model.ConnectorStatusEnable,
		AllowAutoCreateUser: model.ConnectorAutoCreateUserFlagEnable,
		AllowAccountLink:    model.ConnectorAccountLinkFlagEnable,
		SyncProfile:         model.ConnectorSyncProfileFlagEnable,
		EnableTokenStorage:  model.ConnectorTokenStorageFlagEnable,
		Config:              model.ConnectorConfig{},
		ClaimMapping:        model.ConnectorClaimMapping{},
		DomainPolicy:        model.ConnectorDomainPolicy{},
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewConnectorSvc()

	err := svc.Update(ctx, &dtoauth.ConnectorUpdateReq{
		ConnectorID: entity.ID,
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			Name:             "连接器1",
			SyncProfile:      "yes", // 非法枚举值
			AllowAccountLink: model.ConnectorAccountLinkFlagDisable,
		},
	})
	if err == nil {
		t.Fatal("非法开关值应被拒绝")
	}
	if gerror.GetCode(err) != int(code.ConnectorUpdateError) {
		t.Fatalf("期望 ConnectorUpdateError, got %v", err)
	}
	got, err := dao.NewConnectorDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.AllowAccountLink != model.ConnectorAccountLinkFlagEnable {
		t.Fatalf("非法开关值整批不应落库, allow_account_link got %q", got.AllowAccountLink)
	}
}
