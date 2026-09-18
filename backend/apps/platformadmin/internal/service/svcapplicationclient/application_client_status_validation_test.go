package svcapplicationclient

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
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

// TestIsValidApplicationClientPolicies PKCE / auth_time 两个策略白名单本身：
// 只认 enable/disable 与空值（空值表示未提供）。
func TestIsValidApplicationClientPolicies(t *testing.T) {
	for _, p := range []model.ClientPKCEPolicy{"", model.ClientPKCEPolicyEnable, model.ClientPKCEPolicyDisable} {
		if !isValidClientPKCEPolicy(p) {
			t.Fatalf("requirePKCE=%q 应合法", p)
		}
	}
	for _, p := range []model.ClientPKCEPolicy{"true", "false", "on", "off", "1", "0", "ENABLE", "Enable"} {
		if isValidClientPKCEPolicy(p) {
			t.Fatalf("requirePKCE=%q 应非法", p)
		}
	}
	for _, p := range []model.ClientAuthTimeClaimPolicy{"", model.ClientAuthTimeClaimPolicyEnable, model.ClientAuthTimeClaimPolicyDisable} {
		if !isValidClientAuthTimeClaimPolicy(p) {
			t.Fatalf("requireAuthTime=%q 应合法", p)
		}
	}
	for _, p := range []model.ClientAuthTimeClaimPolicy{"true", "false", "on", "off", "1", "0", "ENABLE", "Enable"} {
		if isValidClientAuthTimeClaimPolicy(p) {
			t.Fatalf("requireAuthTime=%q 应非法", p)
		}
	}
}

// TestApplicationClientUpdateRejectsIllegalPolicy 非法 PKCE/auth_time 策略返回功能级错误码且不落库；
// 空串表示未提供，归一为 disable（原 bool 字段在更新中同样按 false 全量写入）。
func TestApplicationClientUpdateRejectsIllegalPolicy(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	entity := &model.ApplicationClientEntity{
		TenantID:        "1",
		AppID:           "app1",
		Code:            "client_policy",
		Name:            "策略客户端",
		Source:          model.ApplicationClientSourceThirdParty,
		Status:          model.ApplicationClientStatusEnable,
		RequirePKCE:     model.ClientPKCEPolicyEnable,
		RequireAuthTime: model.ClientAuthTimeClaimPolicyEnable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	svc := NewApplicationClientSvc()

	err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: entity.ID,
		Name:                "策略客户端",
		RequirePKCE:         "true",
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientUpdateError) {
		t.Fatalf("非法 PKCE 策略应返回 ApplicationClientUpdateError, got %v", err)
	}
	err = svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: entity.ID,
		Name:                "策略客户端",
		RequireAuthTime:     "yes",
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientUpdateError) {
		t.Fatalf("非法 auth_time 策略应返回 ApplicationClientUpdateError, got %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RequirePKCE != model.ClientPKCEPolicyEnable || got.RequireAuthTime != model.ClientAuthTimeClaimPolicyEnable {
		t.Fatalf("非法策略不应落库: %+v", got)
	}

	// 空串表示未提供 → 归一 disable
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: entity.ID,
		Name:                "策略客户端改名",
	}); err != nil {
		t.Fatalf("策略留空应允许: %v", err)
	}
	got, err = dao.NewApplicationClientDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RequirePKCE != model.ClientPKCEPolicyDisable || got.RequireAuthTime != model.ClientAuthTimeClaimPolicyDisable {
		t.Fatalf("空串策略应归一为 disable: %+v", got)
	}

	// 合法值可正常流转
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: entity.ID,
		Name:                "策略客户端改名",
		RequirePKCE:         model.ClientPKCEPolicyEnable,
		RequireAuthTime:     model.ClientAuthTimeClaimPolicyDisable,
	}); err != nil {
		t.Fatalf("合法策略应允许: %v", err)
	}
	got, err = dao.NewApplicationClientDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RequirePKCE != model.ClientPKCEPolicyEnable || got.RequireAuthTime != model.ClientAuthTimeClaimPolicyDisable {
		t.Fatalf("合法策略应生效: %+v", got)
	}
}

// TestApplicationClientCreatePolicyDefaultsToDisable 创建未提供策略时按列默认 disable 落库。
func TestApplicationClientCreatePolicyDefaultsToDisable(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationClientEntity{})

	ctx := newOAuthDeleteCtx("1", "0")
	resp, err := NewApplicationClientSvc().Create(ctx, &dtoapplicationclient.ApplicationClientCreateReq{
		AppID: "app1", Code: "client_default", Name: "缺省策略客户端",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, resp.ApplicationClientID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RequirePKCE != model.ClientPKCEPolicyDisable || got.RequireAuthTime != model.ClientAuthTimeClaimPolicyDisable {
		t.Fatalf("缺省策略应为 disable: %+v", got)
	}
}
