package seed_test

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// 控制台删除菜单的语义：删除即持久删除，种子不得复活。
//
// 实现方式：控制台删除是软删除，软删行仍带 seed_key，即"该内置菜单已被人为下线"的墓碑；
// seedMenus 在创建分支前先查墓碑并跳过。此前种子对缺失行"缺失即重建"，
// 删除会在下次启动被撤销（并丢掉 role_menu 授权），导致控制台根本无法真正删除内置菜单。

// TestSeedIamDoesNotResurrectDeletedMenu 软删一个内置叶子菜单后，重启种子不得重建它。
func TestSeedIamDoesNotResurrectDeletedMenu(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var before int64
	if err := db.Model(&model.MenuEntity{}).Count(&before).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	var menu model.MenuEntity
	if err := db.Where("seed_key = ?", "oauth-client").First(&menu).Error; err != nil {
		t.Fatalf("query menu by seed_key: %v", err)
	}
	if err := db.Delete(&model.MenuEntity{}, "id = ?", menu.ID).Error; err != nil {
		t.Fatalf("soft delete menu: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (3rd) fail: %v", err)
	}

	var resurrected int64
	if err := db.Model(&model.MenuEntity{}).Where("seed_key = ?", "oauth-client").Count(&resurrected).Error; err != nil {
		t.Fatalf("count resurrected menu: %v", err)
	}
	if resurrected != 0 {
		t.Fatalf("被删除的内置菜单被种子复活, count=%d", resurrected)
	}
	var after int64
	if err := db.Model(&model.MenuEntity{}).Count(&after).Error; err != nil {
		t.Fatalf("count menus after: %v", err)
	}
	if after != before-1 {
		t.Fatalf("菜单总数 = %d, want %d（只应少掉被删的那一行）", after, before-1)
	}
	// 墓碑只影响被删的那一项，其余内置菜单照常存在
	var dashboard int64
	if err := db.Model(&model.MenuEntity{}).Where("seed_key = ?", "dashboard").Count(&dashboard).Error; err != nil {
		t.Fatalf("count dashboard: %v", err)
	}
	if dashboard != 1 {
		t.Fatalf("未删除的内置菜单不得受影响, dashboard count=%d", dashboard)
	}
}

// TestSeedIamDoesNotResurrectDeletedMenuSubtree 删除一棵内置子树（目录 + 全部子菜单，
// 等价于控制台的级联删除）后，重启种子不得重建其中任何一行，且不得因父级缺失而中断启动。
func TestSeedIamDoesNotResurrectDeletedMenuSubtree(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	deletedSeedKeys := []string{"grp-platform", "menu"}
	for _, seedKey := range deletedSeedKeys {
		var menu model.MenuEntity
		if err := db.Where("seed_key = ?", seedKey).First(&menu).Error; err != nil {
			t.Fatalf("query menu %s: %v", seedKey, err)
		}
		if err := db.Delete(&model.MenuEntity{}, "id = ?", menu.ID).Error; err != nil {
			t.Fatalf("soft delete menu %s: %v", seedKey, err)
		}
	}

	for i := 0; i < 2; i++ {
		if err := seed.SeedIam(ctx, db); err != nil {
			t.Fatalf("seed (%d) fail: %v", i+2, err)
		}
	}

	for _, seedKey := range deletedSeedKeys {
		var count int64
		if err := db.Model(&model.MenuEntity{}).Where("seed_key = ?", seedKey).Count(&count).Error; err != nil {
			t.Fatalf("count menu %s: %v", seedKey, err)
		}
		if count != 0 {
			t.Errorf("被删除的菜单 %s 被种子复活, count=%d", seedKey, count)
		}
	}
	var dashboard int64
	if err := db.Model(&model.MenuEntity{}).Where("seed_key = ?", "dashboard").Count(&dashboard).Error; err != nil {
		t.Fatalf("count dashboard: %v", err)
	}
	if dashboard != 1 {
		t.Fatalf("未删除的内置菜单不得受影响, dashboard count=%d", dashboard)
	}
}

// TestSeedIamToleratesDeletedMenuParent 父级被删、子菜单仍在（手工删库等非级联路径）时：
// 种子必须跳过子菜单而不是报错中断启动，也不得把它改挂成根菜单。
func TestSeedIamToleratesDeletedMenuParent(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var group model.MenuEntity
	if err := db.Where("seed_key = ?", "grp-platform").First(&group).Error; err != nil {
		t.Fatalf("query grp-platform: %v", err)
	}
	if err := db.Delete(&model.MenuEntity{}, "id = ?", group.ID).Error; err != nil {
		t.Fatalf("soft delete parent: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("父级缺失不得中断启动, got %v", err)
	}

	// 子菜单保持原样：既不被重建，也不被改挂到根节点
	var child model.MenuEntity
	if err := db.Where("seed_key = ?", "menu").First(&child).Error; err != nil {
		t.Fatalf("query child menu: %v", err)
	}
	if child.ParentID != group.ID {
		t.Fatalf("子菜单的父级被种子改写: parent_id=%q, want %q", child.ParentID, group.ID)
	}
	var rootCount int64
	if err := db.Model(&model.MenuEntity{}).Where("seed_key = ? AND parent_id = ?", "menu", "").Count(&rootCount).Error; err != nil {
		t.Fatalf("count root menu: %v", err)
	}
	if rootCount != 0 {
		t.Fatalf("子菜单不得被改挂成根菜单, count=%d", rootCount)
	}
}

// TestSeedIamDoesNotAdoptOperatorMenuWithSeedCode 删除内置菜单后，运营自建一个同 code 的菜单：
// 种子不得把它认领成内置行（回填 seed_key）——否则运维的菜单会被种子接管，删除保护语义随之错乱。
func TestSeedIamDoesNotAdoptOperatorMenuWithSeedCode(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query admin app: %v", err)
	}
	var seedMenu model.MenuEntity
	if err := db.Where("seed_key = ?", "oauth-client").First(&seedMenu).Error; err != nil {
		t.Fatalf("query menu by seed_key: %v", err)
	}
	if err := db.Delete(&model.MenuEntity{}, "id = ?", seedMenu.ID).Error; err != nil {
		t.Fatalf("soft delete seed menu: %v", err)
	}

	operatorMenu := &model.MenuEntity{
		AppID: adminApp.ID, Name: "运维自建客户端", Code: "oauth-client", Path: "/ops-client",
		Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
	}
	if err := db.Create(operatorMenu).Error; err != nil {
		t.Fatalf("create operator menu: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var got model.MenuEntity
	if err := db.Where("id = ?", operatorMenu.ID).First(&got).Error; err != nil {
		t.Fatalf("query operator menu: %v", err)
	}
	if got.SeedKey != "" {
		t.Fatalf("运维自建菜单被种子认领为内置行: seed_key=%q", got.SeedKey)
	}
	if got.Path != "/ops-client" || got.Name != "运维自建客户端" {
		t.Fatalf("运维自建菜单被种子回写: %+v", got)
	}
}

// TestSeedIamRetiredMenuLeavesNoTombstone 退役菜单（版本级下线）走物理删除、不留软删墓碑：
// 否则未来版本重新上线同名 seed_key 时会被墓碑挡住，永远创建不出来。
func TestSeedIamRetiredMenuLeavesNoTombstone(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query admin app: %v", err)
	}
	// 模拟存量库：历史版本种子里有、当前已下线的菜单（retiredMenus 登记为 api-key）
	legacy := &model.MenuEntity{
		AppID: adminApp.ID, SeedKey: "api-key", Name: "API密钥监督", Code: "api-key",
		Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
	}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy menu: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var remaining int64
	if err := db.Unscoped().Model(&model.MenuEntity{}).Where("seed_key = ?", "api-key").Count(&remaining).Error; err != nil {
		t.Fatalf("count retired menu: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("退役菜单必须物理删除（不留墓碑）, count=%d", remaining)
	}
}

// TestSeedIamRetiresLogMenuWithRoleBindings 本批（P4）下线的「审计日志」菜单：存量库首次升级时
// 必须连同 role_menu 授权一并清理——`log` 表与前端页面已删除，留下菜单行就是死链，
// 留下授权则是指向不存在菜单的脏绑定。
//
// 全新库测试测不出这类残留（新库根本没有该行），故这里显式模拟存量库：先建出历史行与授权，
// 再跑种子，断言「菜单物理删除 + role_menu 清零」。
func TestSeedIamRetiresLogMenuWithRoleBindings(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query admin app: %v", err)
	}
	var adminRole model.RoleEntity
	if err := db.Where("app_id = ? AND source = ?", adminApp.ID, model.RoleSourceBuiltin).First(&adminRole).Error; err != nil {
		t.Fatalf("query admin role: %v", err)
	}

	// 模拟存量库：历史版本种子写入的「审计日志」菜单（seed_key=log）+ 内置管理员角色的授权
	legacy := &model.MenuEntity{
		AppID: adminApp.ID, SeedKey: "log", Name: "审计日志", Code: "log", Path: "/log",
		Component: "/log/index", Sort: 2, Type: model.MenuTypeMenu,
		Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
	}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy log menu: %v", err)
	}
	if err := db.Create(&model.RoleMenuEntity{
		TenantID: adminRole.TenantID, RoleID: adminRole.ID, MenuID: legacy.ID,
	}).Error; err != nil {
		t.Fatalf("create legacy role_menu: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var remaining int64
	if err := db.Unscoped().Model(&model.MenuEntity{}).Where("seed_key = ?", "log").Count(&remaining).Error; err != nil {
		t.Fatalf("count retired log menu: %v", err)
	}
	if remaining != 0 {
		t.Errorf("退役的「审计日志」菜单必须物理删除, count=%d", remaining)
	}
	var links int64
	if err := db.Model(&model.RoleMenuEntity{}).Where("menu_id = ?", legacy.ID).Count(&links).Error; err != nil {
		t.Fatalf("count role_menu: %v", err)
	}
	if links != 0 {
		t.Errorf("退役菜单的 role_menu 授权必须一并清理, count=%d", links)
	}
	// 未连带影响其它菜单：菜单管理（platform 的 menu 子项）仍在
	var kept int64
	if err := db.Model(&model.MenuEntity{}).Where("seed_key = ?", "menu").Count(&kept).Error; err != nil {
		t.Fatalf("count menu seed_key: %v", err)
	}
	if kept != 1 {
		t.Errorf("未退役菜单不得受影响, count=%d", kept)
	}
}
