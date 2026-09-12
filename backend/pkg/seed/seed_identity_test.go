package seed_test

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// TestSeedIamKeepsMenuIdentityAfterCodeRename 运营在控制台改掉内置菜单的编码后，重启种子**不得重建一行**：
// 种子身份由 seed_key 承担（= 定义时的 code，创建时写入后不再变化），code 只是可改的业务标识。
//
// 回归背景：引入 seed_key 之前种子按 (app_id, code) 认行，改编码会让种子查不到 → 另建一行，
// 导致菜单重复、role_menu 授权分叉。
func TestSeedIamKeepsMenuIdentityAfterCodeRename(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query admin app: %v", err)
	}
	var before int64
	if err := db.Model(&model.MenuEntity{}).Where("app_id = ?", adminApp.ID).Count(&before).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}

	// 模拟控制台改名：编码 + 名称 + 路径
	var menu model.MenuEntity
	if err := db.Where("seed_key = ?", "log").First(&menu).Error; err != nil {
		t.Fatalf("query menu by seed_key: %v", err)
	}
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", menu.ID).
		Updates(map[string]any{"code": "audit_log", "name": "操作审计", "path": "/audit"}).Error; err != nil {
		t.Fatalf("rename menu: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (3rd) fail: %v", err)
	}

	var after int64
	if err := db.Model(&model.MenuEntity{}).Where("app_id = ?", adminApp.ID).Count(&after).Error; err != nil {
		t.Fatalf("count menus after reseed: %v", err)
	}
	if after != before {
		t.Fatalf("菜单数 = %d, want %d（改编码不得触发种子重建行）", after, before)
	}
	var got model.MenuEntity
	if err := db.Where("id = ?", menu.ID).First(&got).Error; err != nil {
		t.Fatalf("query renamed menu: %v", err)
	}
	if got.Code != "audit_log" || got.Name != "操作审计" || got.Path != "/audit" {
		t.Errorf("改名被种子回写: code=%q name=%q path=%q", got.Code, got.Name, got.Path)
	}
	if got.SeedKey != "log" {
		t.Errorf("seed_key = %q, want log（种子身份键不可变）", got.SeedKey)
	}
	// 被改掉的旧编码不得残留（也不得被种子重新建出来）
	var staleCount int64
	if err := db.Model(&model.MenuEntity{}).Where("app_id = ? AND code = ?", adminApp.ID, "log").
		Count(&staleCount).Error; err != nil {
		t.Fatalf("count stale menu: %v", err)
	}
	if staleCount != 0 {
		t.Errorf("旧编码 log 残留 %d 行，want 0", staleCount)
	}
}

// TestSeedIamKeepsApplicationIdentityAfterCodeRename 应用编码同理由运维掌握：
// 改掉内置应用的 code 后重启种子不得重建应用，菜单/订阅仍挂在同一 app_id 上。
func TestSeedIamKeepsApplicationIdentityAfterCodeRename(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("seed_key = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query app by seed_key: %v", err)
	}
	if err := db.Model(&model.ApplicationEntity{}).Where("id = ?", adminApp.ID).
		Update("code", "platform_console").Error; err != nil {
		t.Fatalf("rename application: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var appCount int64
	if err := db.Model(&model.ApplicationEntity{}).Count(&appCount).Error; err != nil {
		t.Fatalf("count applications: %v", err)
	}
	if appCount != 2 {
		t.Fatalf("应用数 = %d, want 2（改编码不得触发种子重建应用）", appCount)
	}
	var got model.ApplicationEntity
	if err := db.Where("id = ?", adminApp.ID).First(&got).Error; err != nil {
		t.Fatalf("query renamed app: %v", err)
	}
	if got.Code != "platform_console" {
		t.Errorf("应用编码被种子回写: %q", got.Code)
	}
	if got.SeedKey != "platform_admin" {
		t.Errorf("seed_key = %q, want platform_admin", got.SeedKey)
	}
	// 菜单仍挂在这个应用上（改名不该换 id 或改 app_id）
	var menuCount int64
	if err := db.Model(&model.MenuEntity{}).Where("app_id = ?", adminApp.ID).Count(&menuCount).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if menuCount != 11 {
		t.Errorf("应用下菜单数 = %d, want 11", menuCount)
	}
}

// TestSeedIamDoesNotPruneMenuRenamedToRetiredKey 退役清理按 seed_key 认行：
// 运营把一个在用的菜单改名成退役清单里的编码（如 api-key）时，不得被误当成幽灵菜单删掉。
// 兜底按 (app_id, code) 认领只对「尚未回填 seed_key 的历史行」生效（seed_key = ”）。
func TestSeedIamDoesNotPruneMenuRenamedToRetiredKey(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var menu model.MenuEntity
	if err := db.Where("seed_key = ?", "log").First(&menu).Error; err != nil {
		t.Fatalf("query menu by seed_key: %v", err)
	}
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", menu.ID).
		Update("code", "api-key").Error; err != nil {
		t.Fatalf("rename menu to retired code: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	var count int64
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", menu.ID).Count(&count).Error; err != nil {
		t.Fatalf("count menu: %v", err)
	}
	if count != 1 {
		t.Fatalf("改名成退役编码的菜单被误删（count=%d），退役清理必须按 seed_key 认行", count)
	}
}

// TestSeedIamBackfillsSeedKeyForExistingRows 存量库（种子身份键引入之前建的行）首次升级：
// 种子按 (app_id, code) 认领既有行并回填 seed_key，**不新建任何行**，也不改运营已改过的字段。
// 这是升级路径的核心回归——现有部署重启时走的正是这条分支。
func TestSeedIamBackfillsSeedKeyForExistingRows(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	// 模拟存量库：抹掉全部 seed_key（老版本没有该列，等价于列默认值空串）
	if err := db.Model(&model.MenuEntity{}).Where("1 = 1").Update("seed_key", "").Error; err != nil {
		t.Fatalf("clear menu seed_key: %v", err)
	}
	if err := db.Model(&model.ApplicationEntity{}).Where("1 = 1").Update("seed_key", "").Error; err != nil {
		t.Fatalf("clear application seed_key: %v", err)
	}
	var menusBefore, appsBefore int64
	if err := db.Model(&model.MenuEntity{}).Count(&menusBefore).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if err := db.Model(&model.ApplicationEntity{}).Count(&appsBefore).Error; err != nil {
		t.Fatalf("count applications: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (upgrade) fail: %v", err)
	}
	// 再跑一次：回填后必须幂等
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (3rd) fail: %v", err)
	}

	var menusAfter, appsAfter int64
	if err := db.Model(&model.MenuEntity{}).Count(&menusAfter).Error; err != nil {
		t.Fatalf("count menus after: %v", err)
	}
	if err := db.Model(&model.ApplicationEntity{}).Count(&appsAfter).Error; err != nil {
		t.Fatalf("count applications after: %v", err)
	}
	if menusAfter != menusBefore || appsAfter != appsBefore {
		t.Fatalf("回填不得新建行: menu %d->%d, application %d->%d", menusBefore, menusAfter, appsBefore, appsAfter)
	}

	// 每个内置菜单/应用都拿到了与定义一致的 seed_key
	for _, seedKey := range []string{"dashboard", "grp-tenant", "tenant", "log", "department", "tenant-api-key"} {
		var count int64
		if err := db.Model(&model.MenuEntity{}).Where("seed_key = ?", seedKey).Count(&count).Error; err != nil {
			t.Fatalf("count menu seed_key %s: %v", seedKey, err)
		}
		if count != 1 {
			t.Errorf("菜单 seed_key=%s 的行数 = %d, want 1", seedKey, count)
		}
	}
	for _, seedKey := range []string{"platform_admin", "tenant_admin"} {
		var count int64
		if err := db.Model(&model.ApplicationEntity{}).Where("seed_key = ?", seedKey).Count(&count).Error; err != nil {
			t.Fatalf("count application seed_key %s: %v", seedKey, err)
		}
		if count != 1 {
			t.Errorf("应用 seed_key=%s 的行数 = %d, want 1", seedKey, count)
		}
	}
	// 控制台自建行不参与回填（全库没有 seed_key 为空的内置行，但用户自建行仍为空）
	var emptyCount int64
	if err := db.Model(&model.MenuEntity{}).Where("seed_key = ?", "").Count(&emptyCount).Error; err != nil {
		t.Fatalf("count empty seed_key: %v", err)
	}
	if emptyCount != 0 {
		t.Errorf("仍有 %d 行未回填 seed_key（当前库只有内置菜单）", emptyCount)
	}
}
