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
	if err := db.Where("seed_key = ?", "oauth-client").First(&menu).Error; err != nil {
		t.Fatalf("query menu by seed_key: %v", err)
	}
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", menu.ID).
		Updates(map[string]any{"code": "oauth_client_console", "name": "客户端控制台", "path": "/oauth-client-console"}).Error; err != nil {
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
	if got.Code != "oauth_client_console" || got.Name != "客户端控制台" || got.Path != "/oauth-client-console" {
		t.Errorf("改名被种子回写: code=%q name=%q path=%q", got.Code, got.Name, got.Path)
	}
	if got.SeedKey != "oauth-client" {
		t.Errorf("seed_key = %q, want oauth-client（种子身份键不可变）", got.SeedKey)
	}
	// 被改掉的旧编码不得残留（也不得被种子重新建出来）
	var staleCount int64
	if err := db.Model(&model.MenuEntity{}).Where("app_id = ? AND code = ?", adminApp.ID, "oauth-client").
		Count(&staleCount).Error; err != nil {
		t.Fatalf("count stale menu: %v", err)
	}
	if staleCount != 0 {
		t.Errorf("旧编码 oauth-client 残留 %d 行，want 0", staleCount)
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
	if menuCount != 10 {
		t.Errorf("应用下菜单数 = %d, want 10", menuCount)
	}
}
