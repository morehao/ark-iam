//go:build pg

package dbclient

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestTenantScopePluginPostgres 针对本地 PostgreSQL 验证 tenant 插件注入的
// 限定符 SQL 合法（回归测试：golib 插件反引号在 PG 上报 syntax error）。
// 运行方式: go test -tags pg ./pkg/dbclient/ -run TestTenantScopePluginPostgres -v
// 前置条件: 本地 127.0.0.1:5432 存在 postgres/123456 且已建 iam 库。
func TestTenantScopePluginPostgres(t *testing.T) {
	dsn := "postgres://postgres:123456@127.0.0.1:5432/iam?sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	plugin, err := newTenantScopePlugin(tenantScopeSkipTables)
	if err != nil {
		t.Fatalf("newTenantScopePlugin: %v", err)
	}
	if err := db.Use(plugin); err != nil {
		t.Fatalf("use plugin: %v", err)
	}
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 带租户上下文的查询：插件必须注入 "tenant_user"."tenant_id" = ? 且 PG 可执行。
	ctx := context.WithValue(context.Background(), gcontext.KeyTenantID, "seed-tenant")
	var total int64
	if err := db.WithContext(ctx).Model(&model.UserEntity{}).Count(&total).Error; err != nil {
		t.Fatalf("count with tenant ctx fail (backtick regression?): %v", err)
	}

	// 不带租户上下文：插件不注入，查询仍应成功。
	var total2 int64
	if err := db.WithContext(context.Background()).Model(&model.UserEntity{}).Count(&total2).Error; err != nil {
		t.Fatalf("count without tenant ctx fail: %v", err)
	}
	t.Logf("ok: scoped=%d unscoped=%d", total, total2)
}
