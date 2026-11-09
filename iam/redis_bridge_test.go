package iam

import (
	"testing"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"

	"github.com/morehao/ark-iam/pkg/dbclient"
)

// TestConfigRedisBridgesGlobal 校验 Config.Redis 会同步到全局 dbclient.RedisCli，
// 使 OP/SSO 等读全局的共享代码与实例使用同一份连接（无需嵌入方额外设置全局）。
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
