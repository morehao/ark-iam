package server

import (
	"context"
	"errors"
	"testing"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
)

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

	// 预置一张名为 iam_user、但缺 id/created_at 的「三方业务表」（与 IAM 表同名，触发撞名）。
	db := app.DBGetter()(context.Background())
	if err := db.Exec("CREATE TABLE iam_user (external_id TEXT PRIMARY KEY, nickname TEXT)").Error; err != nil {
		t.Fatalf("create conflicting table: %v", err)
	}

	err = app.RunMigrations(context.Background())
	var conflict *ErrTableConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("RunMigrations: want ErrTableConflict, got %v", err)
	}
	if conflict.Table != "iam_user" {
		t.Fatalf("conflict table: want iam_user, got %q", conflict.Table)
	}
}

// TestTableConflictAllowlist 校验白名单可放行冲突表。
func TestTableConflictAllowlist(t *testing.T) {
	app, err := New(Config{
		Env:                 "dev",
		DB:                  DBConfig{DSN: "sqlite://:memory:"},
		Domains:             []Domain{DomainMenu},
		SameDatabase:        true,
		AllowExistingTables: []string{"iam_user"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()
	db := app.DBGetter()(context.Background())
	if err := db.Exec("CREATE TABLE iam_user (external_id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	// 白名单放行：不应返回 ErrTableConflict。
	err = app.RunMigrations(context.Background())
	var conflict *ErrTableConflict
	if errors.As(err, &conflict) {
		t.Fatalf("allowlisted table should not conflict, got %v", err)
	}
}
