package seed_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/seed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDB 打开独立内存 SQLite 并 AutoMigrate 全部 IAM 表（seed 不依赖全局 iam 库注册）。
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:seed_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// TestSeedIamSQLite 在内存 SQLite 上验证种子数据：首次写入 + 二次幂等，
// 并断言管理员用户从属顶级部门（member 关系 + 主归属）。
func TestSeedIamSQLite(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// 第一次：种子写入
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}
	// 第二次：幂等性验证（不应报错、不应重复插入）
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	assertCount := func(tbl string, want int64) {
		t.Helper()
		var n int64
		if err := db.Table(tbl).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		if n != want {
			t.Fatalf("table %s: want %d rows, got %d", tbl, want, n)
		}
	}
	assertCount("tenant", 1)
	assertCount("application", 2)
	assertCount("role", 3)
	assertCount("resource", 2)
	assertCount("scope", 12)
	assertCount("menu", 15)
	assertCount("person", 1)
	assertCount("tenant_user", 1)
	assertCount("application_client", 2)
	assertCount("user_role", 1)
	assertCount("role_menu", 19)
	assertCount("role_scope", 15)
	assertCount("tenant_application", 2)
	assertCount("organization", 1)
	assertCount("organization_user", 1)

	// 管理员从属顶级部门（member 关系 + 主归属）
	var tenant model.TenantEntity
	if err := db.Where("code = ?", "platform").First(&tenant).Error; err != nil {
		t.Fatalf("platform tenant not found: %v", err)
	}
	var adminUser model.UserEntity
	if err := db.Where("tenant_id = ? AND is_owner = ?", tenant.ID, true).First(&adminUser).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}
	var rootOrg model.OrganizationEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&rootOrg).Error; err != nil {
		t.Fatalf("root organization not found: %v", err)
	}
	var ou model.OrganizationUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ? AND organization_id = ? AND relation_type = ? AND is_primary = ?",
		tenant.ID, adminUser.ID, rootOrg.ID, string(model.OrgUserRelationMember), true).First(&ou).Error; err != nil {
		t.Fatalf("admin organization relation not found: %v", err)
	}
}
