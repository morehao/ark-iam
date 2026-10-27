package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/morehao/ark-iam/pkg/model"
)

// newPGDryRunDB 返回一个 PostgreSQL 方言、DryRun 模式、不连库的 *gorm.DB。
//
// DryRun 只渲染 SQL、不执行，因此不需要本地 PG 也能钉住"保留字表名 + Cond 条件"的
// 渲染结果——这正是 `user` 表名最容易回归的地方（手写引号、双重引号、反引号）。
// 真实 PG 的可执行性另由 -tags pg 用例覆盖。
func newPGDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "postgres://dryrun:dryrun@127.0.0.1:1/none?sslmode=disable",
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               logger.Discard,
	})
	if err != nil {
		t.Fatalf("open postgres dry-run: %v", err)
	}
	return db
}

// newCaptureDB 包一层 Session，把 DryRun 渲染出的 SQL 收集到 sqls 里。
func newCaptureDB(t *testing.T) (*gorm.DB, *[]string) {
	t.Helper()
	sqls := make([]string, 0, 4)
	db := newPGDryRunDB(t).Session(&gorm.Session{
		DryRun: true,
		Logger: logger.New(
			// 直接借 gorm 的 logger 接口收集 SQL。
			&sqlRecorder{t: t, sqls: &sqls},
			logger.Config{LogLevel: logger.Info},
		),
	})
	return db, &sqls
}

type sqlRecorder struct {
	t    *testing.T
	sqls *[]string
}

// Printf 实现 logger.Writer：gorm 以 Printf 输出 SQL 文本。
func (r *sqlRecorder) Printf(format string, args ...interface{}) {
	// SQL 行形如 "[0.100ms] [rows:0] SELECT ..."，只留语句本身便于断言。
	line := strings.TrimSpace(fmt.Sprintf(format, args...))
	if idx := strings.Index(line, "SELECT"); idx >= 0 {
		line = line[idx:]
	}
	*r.sqls = append(*r.sqls, line)
}

// TestUserCondRendersQuotedUserTableOnPostgres 钉住 `user` 保留字表名在 PG 方言下的渲染：
// Cond 条件与软删除过滤都必须产出 "user"."col"（表名带引号），且不得出现双重引号或反引号。
func TestUserCondRendersQuotedUserTableOnPostgres(t *testing.T) {
	if model.TableNameUser != "user" {
		t.Fatalf("TableNameUser = %q, want %q", model.TableNameUser, "user")
	}

	db, sqls := newCaptureDB(t)
	userDao := NewUserDao(WithDBGetter(func(context.Context) *gorm.DB { return db }))

	cond := &UserCond{
		BaseCond: nil,
		TenantID: "tenant-1",
		Status:   model.UserStatusActive,
	}
	// GetListByCond 走 deletedScope + BuildCondition，是最能暴露引号问题的路径。
	if _, err := userDao.GetListByCond(context.Background(), cond); err != nil {
		t.Fatalf("GetListByCond (dry-run) fail: %v", err)
	}

	if len(*sqls) == 0 {
		t.Fatal("未捕获到 DryRun SQL")
	}
	sql := (*sqls)[len(*sqls)-1]
	t.Logf("PG dry-run SQL: %s", sql)

	// 契约（golib gormdao/cond.go）：Cond.BuildCondition 收到的 tableName 已是当前方言下的
	// 引用形式（PostgreSQL `"user"`），调用方用它拼前缀即 `"user".col = ?`，不得再手动引用。
	// 列名（tenant_id/status）不是保留字，PG 允许裸列名，因此正确形态是 `"user".tenant_id`。
	wants := []string{
		`FROM "user"`,
		`"user".tenant_id`,
		`"user".status`,
		`"user"."deleted_at"`,
	}
	for _, want := range wants {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL 缺少 %q\n实际: %s", want, sql)
		}
	}

	// 反向断言：不得出现双重引号、反引号（PG 不支持），
	// 也不得出现**未引用**的保留字表名前缀 user.（这正是 tenant_user 改名前的做法）。
	for _, bad := range []string{"`", `""user""`, `"user""`} {
		if strings.Contains(sql, bad) {
			t.Errorf("SQL 出现非法引用 %q（双重引号/反引号）\n实际: %s", bad, sql)
		}
	}
	if strings.Contains(sql, "user.tenant_id") && !strings.Contains(sql, `"user".tenant_id`) {
		t.Errorf("SQL 使用了未引用的保留字表名 user.tenant_id\n实际: %s", sql)
	}
}

// TestUserDaoParamQueryRendersQuotedUserTableOnPostgres 钉住 GetByID 这类不经过 Cond 的路径。
func TestUserDaoParamQueryRendersQuotedUserTableOnPostgres(t *testing.T) {
	db, sqls := newCaptureDB(t)
	userDao := NewUserDao(WithDBGetter(func(context.Context) *gorm.DB { return db }))

	if _, err := userDao.GetByID(context.Background(), "user-1"); err != nil {
		t.Fatalf("GetByID (dry-run) fail: %v", err)
	}
	if len(*sqls) == 0 {
		t.Fatal("未捕获到 DryRun SQL")
	}
	sql := (*sqls)[len(*sqls)-1]
	t.Logf("PG dry-run SQL: %s", sql)
	if !strings.Contains(sql, `FROM "user"`) {
		t.Errorf("GetByID 缺少 FROM \"user\"\n实际: %s", sql)
	}
	if strings.Contains(sql, "`") {
		t.Errorf("GetByID SQL 出现反引号（PG 不支持）\n实际: %s", sql)
	}
}

// TestUserEntityPhysicalDeleteRendersQuotedUserTableOnPostgres 钉住测试夹具的清理路径：
// pkg/testsetup.CleanupTestData 用 db.Unscoped().Where(...).Delete(&model.UserEntity{}) 物理删除，
// 而不是手拼 "DELETE FROM "+表名——后者会生成 `DELETE FROM user` 并在 PG 上语法错误。
func TestUserEntityPhysicalDeleteRendersQuotedUserTableOnPostgres(t *testing.T) {
	// Delete 在 DryRun 下仍会尝试建连，因此换成"只记录不执行"的 connector：
	// 语句照常由 PG 方言渲染，但不会打开真实连接。
	rec := &recordingConn{}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: rec}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open postgres with recording conn: %v", err)
	}
	if err := db.Unscoped().Where("id = ?", "user-1").Delete(&model.UserEntity{}).Error; err != nil {
		t.Fatalf("physical delete fail: %v", err)
	}

	sql := rec.lastSQL()
	t.Logf("PG SQL: %s", sql)
	if !strings.Contains(sql, `DELETE FROM "user"`) {
		t.Errorf("物理删除缺少 DELETE FROM \"user\"\n实际: %s", sql)
	}
	if strings.Contains(sql, "`") {
		t.Errorf("物理删除 SQL 出现反引号（PG 不支持）\n实际: %s", sql)
	}
}

// recordingConn 实现 gorm.Dialector 所需的 ConnPool：只记录 SQL，不连库。
type recordingConn struct {
	mu  sync.Mutex
	sql string
}

func (c *recordingConn) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("recordingConn: not executable")
}

func (c *recordingConn) ExecContext(_ context.Context, query string, _ ...interface{}) (sql.Result, error) {
	c.mu.Lock()
	c.sql = query
	c.mu.Unlock()
	return sql.Result(nonExecResult{}), nil
}

func (c *recordingConn) QueryContext(_ context.Context, query string, _ ...interface{}) (*sql.Rows, error) {
	c.mu.Lock()
	c.sql = query
	c.mu.Unlock()
	return nil, errors.New("recordingConn: not executable")
}

func (c *recordingConn) QueryRowContext(_ context.Context, query string, _ ...interface{}) *sql.Row {
	c.mu.Lock()
	c.sql = query
	c.mu.Unlock()
	return &sql.Row{}
}

func (c *recordingConn) lastSQL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sql
}

type nonExecResult struct{}

func (nonExecResult) LastInsertId() (int64, error) { return 0, nil }
func (nonExecResult) RowsAffected() (int64, error) { return 0, nil }
