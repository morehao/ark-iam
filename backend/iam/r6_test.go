package iam

import (
	"context"
	"errors"
	"testing"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"

	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

// TestDomainDependency 校验域依赖：只选 role 不选 menu 应报错。
func TestDomainDependency(t *testing.T) {
	_, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}, Domains: []Domain{DomainRole}})
	var depErr *ErrDomainDependency
	if !errors.As(err, &depErr) {
		t.Fatalf("New(role only): want ErrDomainDependency, got %v", err)
	}
	// role + menu 应通过。
	if _, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}, Domains: []Domain{DomainRole, DomainMenu}}); err != nil {
		t.Fatalf("New(role+menu): unexpected %v", err)
	}
}

// TestNonDevSecurityValidation 校验 prod 缺安全配置被拒绝。
func TestNonDevSecurityValidation(t *testing.T) {
	_, err := New(Config{
		Env: "prod",
		DB:  DBConfig{DSN: "sqlite://:memory:"},
		Raw: &pkconfig.Config{}, // 缺 cookieSecure/密钥/encryptionKey
	})
	if err == nil {
		t.Fatal("New(prod, insecure config): want error, got nil")
	}
}

// TestTableConflict 校验同库撞名检测：预置一张同名但非 IAM 结构的表 → RunMigrations fail-fast。
func TestTableConflict(t *testing.T) {
	app, err := New(Config{
		Env:          "dev",
		DB:           DBConfig{DSN: "sqlite://:memory:"},
		Domains:      []Domain{DomainMenu},
		SameDatabase: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	// 预置一张名为 user、但缺 id/created_at 的「三方业务表」。
	db := app.DBGetter()(context.Background())
	if err := db.Exec("CREATE TABLE user (external_id TEXT PRIMARY KEY, nickname TEXT)").Error; err != nil {
		t.Fatalf("create conflicting table: %v", err)
	}

	err = app.RunMigrations(context.Background())
	var conflict *ErrTableConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("RunMigrations: want ErrTableConflict, got %v", err)
	}
	if conflict.Table != "user" {
		t.Fatalf("conflict table: want user, got %q", conflict.Table)
	}
}

// TestTableConflictAllowlist 校验白名单可放行冲突表。
func TestTableConflictAllowlist(t *testing.T) {
	app, err := New(Config{
		Env:                 "dev",
		DB:                  DBConfig{DSN: "sqlite://:memory:"},
		Domains:             []Domain{DomainMenu},
		SameDatabase:        true,
		AllowExistingTables: []string{"user"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()
	db := app.DBGetter()(context.Background())
	if err := db.Exec("CREATE TABLE user (external_id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	// 白名单放行：不应返回 ErrTableConflict（后续 AutoMigrate 可能因列冲突另报错，但非撞名错误）。
	err = app.RunMigrations(context.Background())
	var conflict *ErrTableConflict
	if errors.As(err, &conflict) {
		t.Fatalf("allowlisted table should not conflict, got %v", err)
	}
}
