package dao

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newUserApplicationTestDB 打开独立内存 sqlite（不挂租户插件），供本文件验证
// user_application 的索引契约与 DAO 行为：租户作用域由上层用例覆盖，此处不需要。
func newUserApplicationTestDB(t *testing.T) (*gorm.DB, *UserApplicationDao) {
	t.Helper()
	dsn := fmt.Sprintf("file:user_application_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&model.UserApplicationEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db, NewUserApplicationDao(WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
}

// TestUserApplicationIndexDDL 钉住索引形态：唯一索引必须是**部分**索引
// （WHERE deleted_at IS NULL），否则"撤销后重新授予"会撞键。
// 该断言同时证明实体 tag 的 where 选项确实被 migrator 消费（GORM 支持，见
// gorm.io/driver/sqlite migrator 的 `WHERE` 拼接）。
func TestUserApplicationIndexDDL(t *testing.T) {
	db, _ := newUserApplicationTestDB(t)

	indexSQL := map[string]string{}
	rows, err := db.Raw(`SELECT name, COALESCE(sql, '') FROM sqlite_master WHERE type = 'index' AND tbl_name = ?`,
		model.TableNameUserApplication).Rows()
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err != nil {
			t.Fatalf("scan index row: %v", err)
		}
		indexSQL[name] = sql
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate index rows: %v", err)
	}

	uk, ok := indexSQL["uk_user_application_active"]
	if !ok {
		t.Fatalf("缺少唯一索引 uk_user_application_active，实际索引: %v", indexSQL)
	}
	if !strings.Contains(uk, "deleted_at IS NULL") {
		t.Fatalf("uk_user_application_active 必须是部分索引（WHERE deleted_at IS NULL），实际 DDL: %s", uk)
	}
	for _, col := range []string{"tenant_id", "user_id", "app_id"} {
		if !strings.Contains(uk, col) {
			t.Fatalf("uk_user_application_active 缺少列 %s，实际 DDL: %s", col, uk)
		}
	}
	if _, ok := indexSQL["idx_user_application_app"]; !ok {
		t.Fatalf("缺少二级索引 idx_user_application_app，实际索引: %v", indexSQL)
	}
}

// TestUserApplicationUniqueIndexIsPartial 从行为侧再验一次：
// 同 (tenant, user, app) 未删除只允许一行；软删后可重新授予。
func TestUserApplicationUniqueIndexIsPartial(t *testing.T) {
	_, userApplicationDao := newUserApplicationTestDB(t)
	ctx := context.Background()

	first := &model.UserApplicationEntity{
		TenantID: "tenant-1", UserID: "user-1", AppID: "app-1",
		Status: model.UserApplicationStatusEnable, Source: model.UserApplicationSourceManual,
	}
	if err := userApplicationDao.Insert(ctx, first); err != nil {
		t.Fatalf("insert first grant: %v", err)
	}
	dup := &model.UserApplicationEntity{
		TenantID: "tenant-1", UserID: "user-1", AppID: "app-1",
		Status: model.UserApplicationStatusEnable, Source: model.UserApplicationSourceRole,
	}
	if err := userApplicationDao.Insert(ctx, dup); err == nil {
		t.Fatal("同一 (tenant, user, app) 重复插入应被唯一索引拒绝")
	}

	if err := userApplicationDao.Delete(ctx, first.ID, "operator-1"); err != nil {
		t.Fatalf("soft delete first grant: %v", err)
	}
	if err := userApplicationDao.Insert(ctx, dup); err != nil {
		t.Fatalf("软删后重新授予应成功（部分唯一索引），got err: %v", err)
	}
}

// TestUserApplicationCondBatchLookup 钉住门禁与列表共用的批量点查能力：
// 一次查一批 user/app 组合，且 status 过滤生效。
func TestUserApplicationCondBatchLookup(t *testing.T) {
	_, userApplicationDao := newUserApplicationTestDB(t)
	ctx := context.Background()

	seed := []*model.UserApplicationEntity{
		{TenantID: "tenant-1", UserID: "user-1", AppID: "app-1", Status: model.UserApplicationStatusEnable, Source: model.UserApplicationSourceManual},
		{TenantID: "tenant-1", UserID: "user-1", AppID: "app-2", Status: model.UserApplicationStatusDisable, Source: model.UserApplicationSourceRole},
		{TenantID: "tenant-1", UserID: "user-2", AppID: "app-1", Status: model.UserApplicationStatusEnable, Source: model.UserApplicationSourceProvision},
	}
	for _, entity := range seed {
		if err := userApplicationDao.Insert(ctx, entity); err != nil {
			t.Fatalf("insert seed grant: %v", err)
		}
	}

	enabled, err := userApplicationDao.GetListByCond(ctx, &UserApplicationCond{
		TenantID: "tenant-1",
		UserIDs:  []string{"user-1", "user-2"},
		AppIDs:   []string{"app-1"},
		Status:   model.UserApplicationStatusEnable,
	})
	if err != nil {
		t.Fatalf("batch lookup: %v", err)
	}
	if len(enabled) != 2 {
		t.Fatalf("期望 2 条 enable 准入，got %d", len(enabled))
	}

	all, err := userApplicationDao.GetListByCond(ctx, &UserApplicationCond{
		TenantID: "tenant-1",
		UserID:   "user-1",
	})
	if err != nil {
		t.Fatalf("user lookup: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("期望 user-1 共 2 条准入（含 disable），got %d", len(all))
	}

	// 枚举往返：具名类型经 DAO 读出后仍是定义域内的取值
	for _, entity := range all {
		switch entity.Source {
		case model.UserApplicationSourceManual, model.UserApplicationSourceRole, model.UserApplicationSourceProvision:
		default:
			t.Fatalf("source 取值不在定义域内: %q", entity.Source)
		}
		switch entity.Status {
		case model.UserApplicationStatusEnable, model.UserApplicationStatusDisable:
		default:
			t.Fatalf("status 取值不在定义域内: %q", entity.Status)
		}
	}
}
