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

// TestUpdateBuiltInAppMenuAllowsAllFields 内置应用的菜单**字段全部可改**，包括编码与所属应用：
// 种子按 seed_key（种子身份键）认行，不再按 (app_id, code)，因此改名/改归属不会让种子重建行。
//
// 回归背景：编码与归属一度被拒写（种子按 (app_id, code) 查行）；引入 seed_key 后该约束取消。
func TestUpdateBuiltInAppMenuAllowsAllFields(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{})
	app := &model.ApplicationEntity{Code: "platform_admin", Name: "平台管理后台", Source: model.AppSourceBuiltin, Status: model.AppStatusEnable}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	otherApp := &model.ApplicationEntity{Code: "customer_app", Name: "客户应用", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	if err := db.Create(otherApp).Error; err != nil {
		t.Fatalf("seed other application: %v", err)
	}
	menu := &model.MenuEntity{
		AppID: app.ID, SeedKey: "dashboard", Name: "工作台", Code: "dashboard", Path: "/dashboard",
		Icon: "dashboard", Sort: 1, Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember,
		Component: "/dashboard/index", Status: model.MenuStatusEnable,
	}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}

	svc := NewMenuSvc()
	ctx := newGinCtx("1", "0")

	// 一次性改掉全部字段，含编码与所属应用
	if err := svc.Update(ctx, &dtopermission.MenuUpdateReq{
		MenuID: menu.ID,
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: otherApp.ID, Name: "运维自定菜单名", Code: "workbench", Path: "/ops-custom",
			Icon: "SettingOutlined", Sort: 7, Type: model.MenuTypeMenu,
			Visibility: model.MenuVisibilityPublic, Component: "pages/opsCustom", Status: model.MenuStatusDisable,
		},
	}); err != nil {
		t.Fatalf("内置应用菜单字段应全部可改: %v", err)
	}
	got, err := dao.NewMenuDao().GetByID(ctx, menu.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Code != "workbench" || got.AppID != otherApp.ID || got.Name != "运维自定菜单名" ||
		got.Path != "/ops-custom" || got.Component != "pages/opsCustom" || got.Icon != "SettingOutlined" ||
		got.Sort != 7 || got.Visibility != model.MenuVisibilityPublic || got.Status != model.MenuStatusDisable {
		t.Fatalf("展示/结构/编码字段未落库: %+v", got)
	}
	// 种子身份键不受控制台影响（控制台无该字段），改名后种子仍能按它认出这一行
	if got.SeedKey != "dashboard" {
		t.Fatalf("seed_key 不得被控制台改动, got %q", got.SeedKey)
	}

	// 编码为空必须拒绝（应用内唯一由唯一索引兜底）
	if err := svc.Update(ctx, &dtopermission.MenuUpdateReq{
		MenuID: menu.ID,
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: otherApp.ID, Name: "运维自定菜单名", Code: "", Path: "/ops-custom",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityPublic, Status: model.MenuStatusEnable,
		},
	}); err == nil || gerror.GetCode(err) != int(code.MenuUpdateError) {
		t.Fatalf("空编码必须被拒绝, got %v", err)
	}
}

// TestCreateAndDeleteBuiltinAppMenuAllowed 内置应用的菜单同样允许在控制台新增/删除：
// 新增行 seed_key 恒为空（种子只按 seed_key 认领，不会接管运维自建行）；
// 删除内置菜单行留下软删"墓碑"，种子下次启动据此跳过创建（回归见 pkg/seed 的墓碑用例）。
// 回归背景：此前内置应用整棵树禁止增删，"功能扩展/调整"必须改代码发版。
func TestCreateAndDeleteBuiltinAppMenuAllowed(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{}, &model.RoleMenuEntity{})
	builtin := &model.ApplicationEntity{Code: "platform_admin", Name: "平台管理后台", Source: model.AppSourceBuiltin, Status: model.AppStatusEnable}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed builtin application: %v", err)
	}
	svc := NewMenuSvc()
	ctx := newGinCtx("1", "0")

	// 1) 内置应用新增根菜单
	rootResp, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: builtin.ID, Name: "运维新增", Code: "ops-menu",
			Type: model.MenuTypeDirectory, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		},
	})
	if err != nil {
		t.Fatalf("内置应用新增根菜单应放行: %v", err)
	}
	root, err := dao.NewMenuDao().GetByID(ctx, rootResp.MenuID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if root == nil || root.SeedKey != "" {
		t.Fatalf("控制台自建菜单不得带种子身份键: %+v", root)
	}

	// 2) 新增子菜单挂在根菜单下
	childResp, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: builtin.ID, ParentID: rootResp.MenuID, Name: "运维子页", Code: "ops-page",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		},
	})
	if err != nil {
		t.Fatalf("内置应用新增子菜单应放行: %v", err)
	}

	// 3) 删除根菜单：子菜单一并级联删除（否则会留下看不见也删不掉的孤儿行）
	if err := svc.Delete(ctx, &dtopermission.MenuDeleteReq{MenuID: rootResp.MenuID}); err != nil {
		t.Fatalf("内置应用删除菜单应放行: %v", err)
	}
	for _, menuID := range []string{rootResp.MenuID, childResp.MenuID} {
		var count int64
		if err := db.Model(&model.MenuEntity{}).Where("id = ?", menuID).Count(&count).Error; err != nil {
			t.Fatalf("count menu: %v", err)
		}
		if count != 0 {
			t.Fatalf("级联删除后菜单仍可见, menuID:%s count=%d", menuID, count)
		}
	}

	// 4) 删除内置种子菜单：菜单行软删（墓碑）+ role_menu 授权解绑
	seedMenu := &model.MenuEntity{
		AppID: builtin.ID, SeedKey: "dashboard", Name: "工作台", Code: "dashboard",
		Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityMember, Status: model.MenuStatusEnable,
	}
	if err := db.Create(seedMenu).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}
	binding := &model.RoleMenuEntity{TenantID: "1", RoleID: "r1", MenuID: seedMenu.ID}
	if err := db.Create(binding).Error; err != nil {
		t.Fatalf("seed role_menu: %v", err)
	}
	if err := svc.Delete(ctx, &dtopermission.MenuDeleteReq{MenuID: seedMenu.ID}); err != nil {
		t.Fatalf("删除内置菜单应放行: %v", err)
	}
	var active int64
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", seedMenu.ID).Count(&active).Error; err != nil {
		t.Fatalf("count menu: %v", err)
	}
	if active != 0 {
		t.Fatalf("删除未生效, count=%d", active)
	}
	// 软删行仍在（且保留 seed_key）——它是"种子不得复活该菜单"的墓碑证据
	var tombstone int64
	if err := db.Unscoped().Model(&model.MenuEntity{}).
		Where("id = ? AND seed_key = ?", seedMenu.ID, "dashboard").Count(&tombstone).Error; err != nil {
		t.Fatalf("count tombstone: %v", err)
	}
	if tombstone != 1 {
		t.Fatalf("内置菜单删除应留下软删墓碑, count=%d", tombstone)
	}
	var linkCount int64
	if err := db.Model(&model.RoleMenuEntity{}).Where("menu_id = ?", seedMenu.ID).Count(&linkCount).Error; err != nil {
		t.Fatalf("count role_menu: %v", err)
	}
	if linkCount != 0 {
		t.Fatalf("删除菜单必须解绑 role_menu, count=%d", linkCount)
	}
}

// TestMenuParentValidation 上级菜单必须存在、同属一个应用，且不得是自身或其子孙：
// 否则子菜单不会出现在任何应用菜单树里（树从 parent_id="" 构建），成为看不见的孤儿行；
// 成环则整棵子树从树里消失，等于自删。
func TestMenuParentValidation(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.MenuEntity{}, &model.ApplicationEntity{})
	appA := &model.ApplicationEntity{Code: "app_a", Name: "应用A", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	appB := &model.ApplicationEntity{Code: "app_b", Name: "应用B", Source: model.AppSourceThirdParty, Status: model.AppStatusEnable}
	for _, app := range []*model.ApplicationEntity{appA, appB} {
		if err := db.Create(app).Error; err != nil {
			t.Fatalf("seed application: %v", err)
		}
	}
	svc := NewMenuSvc()
	ctx := newGinCtx("1", "0")

	root, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: appA.ID, Name: "根", Code: "root",
			Type: model.MenuTypeDirectory, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		},
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	// 父级不存在
	if _, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: appA.ID, ParentID: "not-exist", Name: "x", Code: "x",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		},
	}); err == nil {
		t.Fatal("父级不存在必须被拒绝")
	}
	// 父级属于别的应用
	if _, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: appB.ID, ParentID: root.MenuID, Name: "y", Code: "y",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		},
	}); err == nil {
		t.Fatal("父级跨应用必须被拒绝")
	}

	child, err := svc.Create(ctx, &dtopermission.MenuCreateReq{
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID: appA.ID, ParentID: root.MenuID, Name: "子", Code: "child",
			Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
		},
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	updateReq := func(parentID string) *dtopermission.MenuUpdateReq {
		return &dtopermission.MenuUpdateReq{
			MenuID: root.MenuID,
			MenuBaseInfo: objpermission.MenuBaseInfo{
				AppID: appA.ID, ParentID: parentID, Name: "根", Code: "root",
				Type: model.MenuTypeDirectory, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
			},
		}
	}
	// 挂到自己的子孙下 → 成环
	if err := svc.Update(ctx, updateReq(child.MenuID)); err == nil {
		t.Fatal("把菜单挂到自己的子孙下必须被拒绝")
	}
	// 挂到自己下
	if err := svc.Update(ctx, updateReq(root.MenuID)); err == nil {
		t.Fatal("把菜单挂到自己下必须被拒绝")
	}
}
