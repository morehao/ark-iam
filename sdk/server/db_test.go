package server

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

// TestDBInjection 校验 App 持有可注入的连接与租户插件生效。
func TestDBInjection(t *testing.T) {
	app, err := New(Config{Env: "dev", DB: DBConfig{DSN: "sqlite://:memory:"}})
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
	if err := db.AutoMigrate(&model.UserEntity{}); err != nil {
		t.Fatalf("AutoMigrate(UserEntity): %v", err)
	}
	// 带租户作用域查询应正常。
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

// TestRedisOptional 校验未配置 Redis 时 App 可正常构造、Redis() 为 nil。
func TestRedisOptional(t *testing.T) {
	app, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()
	if app.Redis() != nil {
		t.Fatal("Redis() should be nil when not configured")
	}
}

// TestConfigRedisBridgesGlobal 校验 Config.Redis 会同步到全局 dbclient.RedisCli。
func TestConfigRedisBridgesGlobal(t *testing.T) {
	prev := dbclient.RedisCli
	defer func() { dbclient.RedisCli = prev }()
	dbclient.RedisCli = nil

	app, err := New(Config{
		Env:   "dev",
		DB:    DBConfig{DSN: "sqlite://:memory:"},
		Redis: &RedisConfig{Addr: "127.0.0.1:6379"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()
	if dbclient.RedisCli == nil {
		t.Fatal("Config.Redis should bridge to global dbclient.RedisCli, got nil")
	}
	if app.Redis() == nil {
		t.Fatal("app.Redis() nil")
	}
}

// TestNoRedisLeavesGlobalNil 校验未配置 Redis 时不错误设置全局。
func TestNoRedisLeavesGlobalNil(t *testing.T) {
	prev := dbclient.RedisCli
	defer func() { dbclient.RedisCli = prev }()
	dbclient.RedisCli = nil

	app, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()
	if dbclient.RedisCli != nil {
		t.Fatal("no Redis config should leave global nil")
	}
}
