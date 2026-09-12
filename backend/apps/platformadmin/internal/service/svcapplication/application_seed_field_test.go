package svcapplication

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gerror"
)

// TestUpdateBuiltInApplicationAllowsDisplayFields 内置应用的名称/描述归运维（字段权威矩阵
// create_only）：控制台可改，且重启不被种子回写。
//
// 回归背景：这两者一度是 reconcile（种子唯一写者），控制台直接拒写，页面完全不能改；
// reconcile 收窄到「定位键 + 安全不变式」后，种子的值只是创建时的初值
// （见 pkg/seed 的 TestSeedIamMigratesLegacyNamesOnly）。
func TestUpdateBuiltInApplicationAllowsDisplayFields(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	builtin := &model.ApplicationEntity{
		Code:        "platform_admin",
		Name:        "平台管理后台",
		Description: "平台管理后台应用",
		Source:      model.AppSourceBuiltin,
		Status:      model.AppStatusEnable,
	}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	svc := NewApplicationSvc()
	ctx := newDeleteCtx("0")

	// 内置应用改名/改描述/改启停排序：全部放行
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:       builtin.ID,
		Name:        "运维自定名",
		Description: "运维自定描述",
		Status:      model.AppStatusDisable,
		Sort:        9,
	}); err != nil {
		t.Fatalf("内置应用的展示字段应可改: %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "运维自定名" || got.Description != "运维自定描述" {
		t.Fatalf("展示字段未落库: name=%q description=%q", got.Name, got.Description)
	}
	if got.Status != model.AppStatusDisable || got.Sort != 9 {
		t.Fatalf("启停/排序未落库: status=%q sort=%d", got.Status, got.Sort)
	}
	// source（安全不变式）不在 Update 请求里，控制台无法改写内置标记
	if got.Source != model.AppSourceBuiltin {
		t.Fatalf("source 不得被控制台改写: %q", got.Source)
	}

	// 第三方应用同样放行
	thirdParty := &model.ApplicationEntity{Code: "customer_app", Name: "客户应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	if err := db.Create(thirdParty).Error; err != nil {
		t.Fatalf("seed third party application: %v", err)
	}
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID: thirdParty.ID, Name: "客户应用改名", Description: "新描述", Status: model.AppStatusEnable,
	}); err != nil {
		t.Fatalf("第三方应用改名应放行: %v", err)
	}
}

// TestUpdateBuiltInApplicationRejectsCodeRename 内置应用（source=builtin）的编码保持只读：
// 控制台菜单入口仍按该编码定位（svcpermission.MyTree 按 platform_admin 查应用、
// tenantadmin loadConsoleApps 只保留 tenant_admin），改名会当场让对应控制台侧边栏失联。
// 其余字段照常可改（见 TestUpdateBuiltInApplicationAllowsDisplayFields）。
func TestUpdateBuiltInApplicationRejectsCodeRename(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	builtin := &model.ApplicationEntity{
		Code: "platform_admin", SeedKey: "platform_admin", Name: "平台管理后台",
		Source: model.AppSourceBuiltin, Status: model.AppStatusEnable,
	}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	svc := NewApplicationSvc()
	ctx := newDeleteCtx("0")

	err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID: builtin.ID, Code: "platform_console", Name: builtin.Name, Status: builtin.Status,
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationBuiltInCodeImmutableError) {
		t.Fatalf("内置应用改编码必须被拒绝, got %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Code != "platform_admin" {
		t.Fatalf("被拒的改名不得落库: %q", got.Code)
	}

	// 回传未变化的编码不受影响（幂等友好），其它字段照常可改
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID: builtin.ID, Code: "platform_admin", Name: "运维自定名", Status: builtin.Status,
	}); err != nil {
		t.Fatalf("编码未变化时其余字段应可改: %v", err)
	}
}

// TestUpdateApplicationAllowsCodeRename 自建应用的编码可改：种子按 seed_key 认行，
// 改名不会导致重建应用；菜单/订阅/角色都挂 app_id，不受影响。
func TestUpdateApplicationAllowsCodeRename(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	thirdParty := &model.ApplicationEntity{
		Code: "customer_app", Name: "客户应用",
		Source: model.AppSourceThirdParty, Status: model.AppStatusEnable,
	}
	if err := db.Create(thirdParty).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	svc := NewApplicationSvc()
	ctx := newDeleteCtx("0")

	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID: thirdParty.ID, Code: "customer_console", Name: thirdParty.Name, Status: thirdParty.Status,
	}); err != nil {
		t.Fatalf("自建应用改编码应放行: %v", err)
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, thirdParty.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Code != "customer_console" {
		t.Fatalf("编码未落库: %q", got.Code)
	}
}

// TestUpdateApplicationRejectsInvalidCode 非法的应用编码（连字符/数字开头/大写）必须被拒，
// 且不落库——与创建时同一套规则（model.AppCodePattern）。
func TestUpdateApplicationRejectsInvalidCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	app := &model.ApplicationEntity{Code: "my_app", Name: "应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	svc := NewApplicationSvc()
	ctx := newDeleteCtx("0")

	for _, badCode := range []string{"my-app", "My_App", "1app", "app.web", "app web"} {
		err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
			AppID: app.ID, Code: badCode, Name: app.Name, Status: app.Status,
		})
		if err == nil {
			t.Fatalf("非法编码 %q 必须被拒绝", badCode)
		}
		if gerror.GetCode(err) != int(code.ApplicationCodeInvalidError) {
			t.Fatalf("编码 %q 期望 ApplicationCodeInvalidError, got %v", badCode, err)
		}
	}
	got, err := dao.NewApplicationDao().GetByID(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Code != "my_app" {
		t.Fatalf("被拒的改名不得落库: %q", got.Code)
	}
}
