package iam

import (
	"context"
	"errors"
	"testing"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
)

// TestNewRequiresDSN 校验空 DSN 被拒绝。
func TestNewRequiresDSN(t *testing.T) {
	_, err := New(Config{})
	if !errors.Is(err, ErrDBDSNRequired) {
		t.Fatalf("New(empty DSN): want ErrDBDSNRequired, got %v", err)
	}
}

// TestR1DBInjection 校验 App 持有可注入的连接与租户插件生效。
//
// 用内存 sqlite 建 person 表（person 在 tenantScopeSkipTables 里、无 tenant_id，
// 用 user 表更能验证插件注入，但 user 有 tenant_id 需建列——这里仅验证连接可用 +
// person 可读写，插件生效的 fail-closed 用例在 R6 撞名/隔离测试里补）。
func TestR1DBInjection(t *testing.T) {
	app, err := New(Config{
		Env: "dev",
		DB:  DBConfig{DSN: "sqlite://:memory:"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	if app.DBGetter() == nil {
		t.Fatal("DBGetter() is nil")
	}

	ctx := context.Background()
	db := app.DBGetter()(ctx)
	if db == nil {
		t.Fatal("dbGetter(ctx) returned nil")
	}

	// 建一张最小 user 表（含 tenant_id），验证插件注入路径不报错的前提成立。
	if err := db.AutoMigrate(&model.UserEntity{}); err != nil {
		t.Fatalf("AutoMigrate(UserEntity): %v", err)
	}

	// 带租户作用域查询应正常（不返回 ErrTenantScopeMissing）。
	scoped := dbclient.CurrentTenantContext(ctx, "tenant-x")
	var count int64
	if err := app.DBGetter()(scoped).Model(&model.UserEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count with tenant scope: %v", err)
	}

	// 未声明作用域应 fail-closed（插件生效的直接证据）。
	err = app.DBGetter()(context.Background()).Model(&model.UserEntity{}).Count(&count).Error
	if !errors.Is(err, dbclient.ErrTenantScopeMissing) {
		t.Fatalf("count without scope: want ErrTenantScopeMissing, got %v", err)
	}
}

// TestR1RedisOptional 校验未配置 Redis 时 App 可正常构造、Redis() 为 nil。
func TestR1RedisOptional(t *testing.T) {
	app, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()
	if app.Redis() != nil {
		t.Fatal("Redis() should be nil when not configured")
	}
}

