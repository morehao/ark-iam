//go:build pg

package seed_test

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/seed"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestSeedIamAgainstPostgres 针对本地 PostgreSQL 验证 AutoMigrate + 种子数据的幂等性。
// 运行方式: go test -tags pg ./pkg/seed/ -run TestSeedIamAgainstPostgres -v
// 前置条件: 本地 127.0.0.1:5432 存在 postgres/postgres 且已创建 iam 库。
func TestSeedIamAgainstPostgres(t *testing.T) {
	dsn := "postgres://postgres:123456@127.0.0.1:5432/iam?sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	// 清理旧数据（顺序：先删关联表，再删主表）
	cleanup := func() {
		tables := []string{
			"user_role", "role_menu", "tenant_application",
			"application_client_secret", "application_client", "menu",
			"role", "user_identity", "user_login_log",
			"organization_user", "organization", "tenant_user", "person",
			"refresh_token", "session", "audit_log", "api_key", "connector",
			"domain", "system", "log", "application", "tenant",
		}
		for _, tbl := range tables {
			_ = db.Exec("DROP TABLE IF EXISTS " + tbl).Error
		}
	}

	cleanup()

	// 第一次：AutoMigrate + Seed
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate fail: %v", err)
	}
	if err := seed.SeedIam(context.Background(), db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	// 第二次：幂等性验证（不应报错、不应重复插入）
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate (2nd) fail: %v", err)
	}
	if err := seed.SeedIam(context.Background(), db); err != nil {
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
	assertCount("role", 1)
	assertCount("menu", 13)
	assertCount("person", 1)
	assertCount("tenant_user", 1)
	assertCount("application_client", 2)
	assertCount("user_role", 1)
	assertCount("role_menu", 13)
	assertCount("tenant_application", 2)
	assertCount("organization", 1)
	assertCount("organization_user", 1)

	// 验证管理员用户归属
	var u model.UserEntity
	if err := db.Where("is_owner = ?", true).First(&u).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}
	if u.TenantID == "" || u.PersonID == "" {
		t.Fatalf("admin user missing tenant/person linkage: %+v", u)
	}
	t.Logf("admin user id=%s tenant=%s person=%s", u.ID, u.TenantID, u.PersonID)

	// 验证管理员从属顶级部门（member 关系 + 主归属）
	var rootOrg model.OrganizationEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", u.TenantID, "").First(&rootOrg).Error; err != nil {
		t.Fatalf("root organization not found: %v", err)
	}
	var ou model.OrganizationUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ? AND organization_id = ? AND relation_type = ? AND is_primary = ?",
		u.TenantID, u.ID, rootOrg.ID, string(model.OrgUserRelationMember), true).First(&ou).Error; err != nil {
		t.Fatalf("admin organization relation not found: %v", err)
	}
	t.Logf("admin org relation: org=%s relation=%s primary=%v", ou.OrganizationID, ou.RelationType, ou.IsPrimary)

	cleanup()
	t.Log("PG AutoMigrate + Seed idempotency check passed")
}
