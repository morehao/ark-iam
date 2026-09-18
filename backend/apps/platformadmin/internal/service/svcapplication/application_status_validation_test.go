package svcapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
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

// TestIsValidAppPolicy 两个策略白名单本身：只认 enable/disable 与空值（空值语义为「不修改」）。
func TestIsValidAppPolicy(t *testing.T) {
	for _, p := range []model.AppPersonCreateTenantPolicy{"", model.AppPersonCreateTenantPolicyEnable, model.AppPersonCreateTenantPolicyDisable} {
		if !isValidPersonCreateTenantPolicy(p) {
			t.Fatalf("personCreateTenant=%q 应合法", p)
		}
	}
	for _, p := range []model.AppPersonCreateTenantPolicy{"true", "false", "on", "off", "1", "0", "ENABLE", "Enable"} {
		if isValidPersonCreateTenantPolicy(p) {
			t.Fatalf("personCreateTenant=%q 应非法", p)
		}
	}
	for _, p := range []model.AppJoinByInvitePolicy{"", model.AppJoinByInvitePolicyEnable, model.AppJoinByInvitePolicyDisable} {
		if !isValidJoinByInvitePolicy(p) {
			t.Fatalf("joinByInvite=%q 应合法", p)
		}
	}
	for _, p := range []model.AppJoinByInvitePolicy{"true", "false", "on", "off", "1", "0", "ENABLE", "Enable"} {
		if isValidJoinByInvitePolicy(p) {
			t.Fatalf("joinByInvite=%q 应非法", p)
		}
	}
}

// TestApplicationUpdateRejectsIllegalPolicy 校验白名单已接入 service 入口：
// 非法策略返回功能级错误码且不落库；空串表示不修改，合法值可正常流转。
func TestApplicationUpdateRejectsIllegalPolicy(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	entity := &model.ApplicationEntity{
		Code:                    "app_policy",
		Name:                    "策略应用",
		Source:                  model.AppSourceThirdParty,
		Status:                  model.AppStatusEnable,
		AllowPersonCreateTenant: model.AppPersonCreateTenantPolicyEnable,
		AllowJoinByInvite:       model.AppJoinByInvitePolicyDisable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newApplicationCtx()
	svc := NewApplicationSvc()

	err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:                   entity.ID,
		Name:                    "策略应用",
		AllowPersonCreateTenant: "true",
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationUpdateError) {
		t.Fatalf("非法个人建租户策略应返回 ApplicationUpdateError, got %v", err)
	}
	err = svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:             entity.ID,
		Name:              "策略应用",
		AllowJoinByInvite: "yes",
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationUpdateError) {
		t.Fatalf("非法邀请加入策略应返回 ApplicationUpdateError, got %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.AllowPersonCreateTenant != model.AppPersonCreateTenantPolicyEnable ||
		got.AllowJoinByInvite != model.AppJoinByInvitePolicyDisable {
		t.Fatalf("非法策略不应落库: %+v", got)
	}

	// 空串表示不修改；合法值可正常流转
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{AppID: entity.ID, Name: "策略应用改名"}); err != nil {
		t.Fatalf("策略留空应允许: %v", err)
	}
	got, err = dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.AllowPersonCreateTenant != model.AppPersonCreateTenantPolicyEnable ||
		got.AllowJoinByInvite != model.AppJoinByInvitePolicyDisable {
		t.Fatalf("留空不应覆盖策略: %+v", got)
	}
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:                   entity.ID,
		Name:                    "策略应用改名",
		AllowPersonCreateTenant: model.AppPersonCreateTenantPolicyDisable,
		AllowJoinByInvite:       model.AppJoinByInvitePolicyEnable,
	}); err != nil {
		t.Fatalf("合法策略应允许: %v", err)
	}
	got, err = dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.AllowPersonCreateTenant != model.AppPersonCreateTenantPolicyDisable ||
		got.AllowJoinByInvite != model.AppJoinByInvitePolicyEnable {
		t.Fatalf("合法策略应生效: %+v", got)
	}
}

// TestApplicationCreatePolicyDefaultsToDisable 创建未提供策略时按列默认 disable 落库
// （原 *bool 缺省同样等价于关闭）。
func TestApplicationCreatePolicyDefaultsToDisable(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{})

	ctx := newApplicationCtx()
	resp, err := NewApplicationSvc().Create(ctx, &dtoapplication.ApplicationCreateReq{Code: "app_default", Name: "缺省策略应用"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, resp.AppID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.AllowPersonCreateTenant != model.AppPersonCreateTenantPolicyDisable ||
		got.AllowJoinByInvite != model.AppJoinByInvitePolicyDisable {
		t.Fatalf("缺省策略应为 disable: %+v", got)
	}
}

// TestApplicationCreateRejectsIllegalPolicy 创建路径同样拦截非法策略且不落库。
func TestApplicationCreateRejectsIllegalPolicy(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})

	ctx := newApplicationCtx()
	_, err := NewApplicationSvc().Create(ctx, &dtoapplication.ApplicationCreateReq{
		Code:                    "app_bad_policy",
		Name:                    "非法策略应用",
		AllowPersonCreateTenant: "true",
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationCreateError) {
		t.Fatalf("非法策略应返回 ApplicationCreateError, got %v", err)
	}
	var count int64
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "app_bad_policy").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("非法策略不应创建应用, count=%d", count)
	}
}
