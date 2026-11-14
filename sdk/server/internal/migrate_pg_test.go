//go:build pg

package internal

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
)

// 复用 pkg/seed 的 PG 测试约定（见 pkg/seed/seed_pg_test.go）：-tags pg 构建标签 +
// IAM_PG_TEST_DSN 环境变量 + 「库名必须含 test」的破坏性保护。本包内自建一份，避免
// 跨包依赖（seed 是 seed_test 外部测试包，无法复用其未导出 helper）。
const defaultPGMigrateDSN = "postgres://postgres:123456@127.0.0.1:5432/iam_migratetest?sslmode=disable"

// pgMigrateTestDSN 取测试 DSN（可用 IAM_PG_TEST_DSN 覆盖），并强制目标库名含 test——
// 本用例会 DROP TABLE，误指开发库会清掉本地数据。
func pgMigrateTestDSN(t *testing.T) string {
	t.Helper()
	dsn := defaultPGMigrateDSN
	if v := os.Getenv("IAM_PG_TEST_DSN"); v != "" {
		dsn = v
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if !strings.Contains(dbName, "test") {
		t.Fatalf("拒绝在非测试库 %q 上执行破坏性用例：请把 IAM_PG_TEST_DSN 指向库名含 test 的专用库", dbName)
	}
	return dsn
}

// newPGApp 构造一个同库模式的 App（撞名检测生效），并注册 Close 清理。
func newPGApp(t *testing.T, dsn string, allow []string) *App {
	t.Helper()
	app, err := New(Config{
		Env:                 "test",
		DB:                  DBConfig{DSN: dsn},
		Domains:             []Domain{DomainMenu},
		SameDatabase:        true,
		AllowExistingTables: allow,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { app.Close() })
	return app
}

// dropIAMTables 清理本包用例可能建出的 IAM 表，避免跨用例残留污染。
// 只删本包用到的表（这里用全量 iamTableNames 兜底，DropIfExists 幂等）。
func dropIAMTables(t *testing.T, app *App) {
	t.Helper()
	db := app.DBGetter()(context.Background())
	names, err := iamTableNames(db)
	if err != nil {
		t.Fatalf("resolve table names: %v", err)
	}
	for _, name := range names {
		if err := db.Exec(`DROP TABLE IF EXISTS "` + name + `" CASCADE`).Error; err != nil {
			t.Fatalf("drop table %s: %v", name, err)
		}
	}
}

// TestTableConflictPostgres 校验 PG 上同库撞名检测：预置一张同名但非 IAM 结构的表
// → RunMigrations fail-fast 抛 ErrTableConflict（Table==iam_user）。
func TestTableConflictPostgres(t *testing.T) {
	dsn := pgMigrateTestDSN(t)
	app := newPGApp(t, dsn, nil)
	db := app.DBGetter()(context.Background())
	dropIAMTables(t, app)

	if err := db.Exec("CREATE TABLE iam_user (external_id TEXT PRIMARY KEY, nickname TEXT)").Error; err != nil {
		t.Fatalf("create conflicting: %v", err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS iam_user CASCADE") })

	err := app.RunMigrations(context.Background())
	var conflict *ErrTableConflict
	if !errors.As(err, &conflict) || conflict.Table != "iam_user" {
		t.Fatalf("want ErrTableConflict(iam_user), got %v", err)
	}
}

// TestTableConflictNormalIAMTablePostgres 校验 PG 上「正常建表后二次迁移不误报」：
// 先正常 RunMigrations 建全表（含 id + created_at），再跑一次 → 不应报 ErrTableConflict。
func TestTableConflictNormalIAMTablePostgres(t *testing.T) {
	dsn := pgMigrateTestDSN(t)
	app := newPGApp(t, dsn, nil)
	dropIAMTables(t, app)
	t.Cleanup(func() { dropIAMTables(t, app) })

	if err := app.RunMigrations(context.Background()); err != nil {
		t.Fatalf("first RunMigrations: %v", err)
	}
	err := app.RunMigrations(context.Background())
	var conflict *ErrTableConflict
	if errors.As(err, &conflict) {
		t.Fatalf("normal IAM tables should not conflict on re-migrate, got %v", err)
	}
}

// TestTableConflictAllowlistPostgres 校验 PG 上白名单放行：iam_user 缺 id/created_at
// 但列入 AllowExistingTables → 不冲突。
func TestTableConflictAllowlistPostgres(t *testing.T) {
	dsn := pgMigrateTestDSN(t)
	app := newPGApp(t, dsn, []string{"iam_user"})
	db := app.DBGetter()(context.Background())
	dropIAMTables(t, app)

	if err := db.Exec("CREATE TABLE iam_user (external_id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS iam_user CASCADE") })

	err := app.RunMigrations(context.Background())
	var conflict *ErrTableConflict
	if errors.As(err, &conflict) {
		t.Fatalf("allowlisted table should not conflict, got %v", err)
	}
}
