package svcpermission

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gerror"
)

// TestUpdateBuiltInAppMenuRejectsSeedOwnedFields 内置应用的菜单树归平台版本定义：
// 结构/展示字段（name/code/path/icon/sort/component/type/visibility/parent）控制台拒写，
// status 归运维可改。回归背景：seedMenus 会收敛这些字段，此前控制台可写 → 菜单改名重启被回滚。
func TestUpdateBuiltInAppMenuRejectsSeedOwnedFields(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{})
	app := &model.ApplicationEntity{Code: "platform_admin", Name: "平台管理后台", Source: model.AppSourceBuiltin, Status: model.AppStatusEnable}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	menu := &model.MenuEntity{
		AppID: app.ID, Name: "工作台", Code: "dashboard", Path: "/dashboard", Icon: "dashboard",
		Sort: 1, Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember,
		Component: "/dashboard/index", Status: model.MenuStatusEnable,
	}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}

	svc := NewMenuSvc()
	ctx := newGinCtx("1", "0")
	baseReq := func(name string, status model.MenuStatus) *dtopermission.MenuUpdateReq {
		return &dtopermission.MenuUpdateReq{
			MenuID: menu.ID,
			MenuBaseInfo: objpermission.MenuBaseInfo{
				AppID: app.ID, Name: name, Code: "dashboard", Path: "/dashboard", Icon: "dashboard",
				Sort: 1, Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember,
				Component: "/dashboard/index", Status: status,
			},
		}
	}

	err := svc.Update(ctx, baseReq("运维自定名", model.MenuStatusEnable))
	if err == nil {
		t.Fatal("内置应用菜单改名必须被拒绝")
	}
	if gerror.GetCode(err) != int(code.MenuBuiltInFieldImmutableError) {
		t.Fatalf("期望 MenuBuiltInFieldImmutableError, got %v", err)
	}
	got, err := dao.NewMenuDao().GetByID(ctx, menu.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "工作台" {
		t.Fatalf("被拒的改名不得落库, got %q", got.Name)
	}

	// status 归运维：只改状态应放行
	if err := svc.Update(ctx, baseReq("工作台", model.MenuStatusDisable)); err != nil {
		t.Fatalf("内置应用菜单的状态应可改: %v", err)
	}
	got, err = dao.NewMenuDao().GetByID(ctx, menu.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != model.MenuStatusDisable {
		t.Fatalf("状态未落库: %q", got.Status)
	}

	// 非内置应用的菜单不受该规则约束
	thirdApp := &model.ApplicationEntity{Code: "customer_app", Name: "客户应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	if err := db.Create(thirdApp).Error; err != nil {
		t.Fatalf("seed third party application: %v", err)
	}
	customMenu := &model.MenuEntity{
		AppID: thirdApp.ID, Name: "客户页面", Code: "customer-page", Path: "/customer-page",
		Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember, Status: model.MenuStatusEnable,
	}
	if err := db.Create(customMenu).Error; err != nil {
		t.Fatalf("seed custom menu: %v", err)
	}
	if err := svc.Update(ctx, &dtopermission.MenuUpdateReq{
		MenuID: customMenu.ID,
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: thirdApp.ID, Name: "客户页面改名", Code: "customer-page", Path: "/customer-page",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember, Status: model.MenuStatusEnable,
		},
	}); err != nil {
		t.Fatalf("第三方应用菜单改名应放行: %v", err)
	}
}

// TestCreateAndDeleteBuiltinAppMenuRejected 内置应用的菜单树随平台版本交付：
// 新增（新增后既改不了也删不掉）与删除（下次启动会被重新播种并丢授权）都必须拒写；
// 非内置应用的新增/删除不受约束。
func TestCreateAndDeleteBuiltinAppMenuRejected(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{})
	builtin := &model.ApplicationEntity{Code: "platform_admin", Name: "平台管理后台", Source: model.AppSourceBuiltin, Status: model.AppStatusEnable}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed builtin application: %v", err)
	}
	svc := NewMenuSvc()
	ctx := newGinCtx("1", "0")

	_, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: builtin.ID, Name: "运维新增", Code: "ops-menu",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		},
	})
	if err == nil || gerror.GetCode(err) != int(code.MenuBuiltInFieldImmutableError) {
		t.Fatalf("内置应用新增菜单必须被拒绝, got %v", err)
	}

	seedMenu := &model.MenuEntity{
		AppID: builtin.ID, Name: "工作台", Code: "dashboard",
		Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember, Status: model.MenuStatusEnable,
	}
	if err := db.Create(seedMenu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}
	if err := svc.Delete(ctx, &dtopermission.MenuDeleteReq{MenuID: seedMenu.ID}); err == nil ||
		gerror.GetCode(err) != int(code.MenuBuiltInFieldImmutableError) {
		t.Fatalf("内置应用删除菜单必须被拒绝, got %v", err)
	}
	var count int64
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", seedMenu.ID).Count(&count).Error; err != nil {
		t.Fatalf("count menu: %v", err)
	}
	if count != 1 {
		t.Fatalf("被拒的删除不得落库, count=%d", count)
	}

	// 非内置应用：新增与删除均放行
	thirdApp := &model.ApplicationEntity{Code: "customer_app", Name: "客户应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	if err := db.Create(thirdApp).Error; err != nil {
		t.Fatalf("seed third party application: %v", err)
	}
	resp, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: thirdApp.ID, Name: "客户页面", Code: "customer-page",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember, Status: model.MenuStatusEnable,
		},
	})
	if err != nil {
		t.Fatalf("第三方应用新增菜单应放行: %v", err)
	}
	if err := svc.Delete(ctx, &dtopermission.MenuDeleteReq{MenuID: resp.MenuID}); err != nil {
		t.Fatalf("第三方应用删除菜单应放行: %v", err)
	}
}
