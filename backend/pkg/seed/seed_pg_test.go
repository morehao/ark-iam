//go:build pg

package seed_test

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// defaultPGTestDSN 默认指向专用测试库，绝不复用开发库 iam（本用例会 DROP 其中的业务表）。
const defaultPGTestDSN = "postgres://postgres:123456@127.0.0.1:5432/iam_seedtest?sslmode=disable"

// pgTestDSN 取测试 DSN（可用 IAM_PG_TEST_DSN 覆盖），并强制目标库名含 test——
// 本用例执行 DROP TABLE，误指开发库会清掉本地数据。
func pgTestDSN(t *testing.T) string {
	t.Helper()
	dsn := defaultPGTestDSN
	if v := os.Getenv("IAM_PG_TEST_DSN"); v != "" {
		dsn = v
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if !strings.Contains(dbName, "test") {
		t.Fatalf("拒绝在非测试库 %q 上执行破坏性种子用例：请把 IAM_PG_TEST_DSN 指向专用测试库（库名含 test）", dbName)
	}
	return dsn
}

// TestSeedIamAgainstPostgres 针对本地 PostgreSQL 验证 AutoMigrate + 种子数据的幂等性。
//
// 运行方式: go test -tags pg ./pkg/seed/ -run TestSeedIamAgainstPostgres -v
// 前置条件: 本地 127.0.0.1:5432 存在 postgres/<pwd> 且已创建测试库（默认 iam_seedtest）：
//
//	docker exec postgres18 psql -U postgres -c "CREATE DATABASE iam_seedtest;"
func TestSeedIamAgainstPostgres(t *testing.T) {
	db, err := gorm.Open(postgres.Open(pgTestDSN(t)), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	// 清理旧数据（顺序：先删关联表，再删主表）
	cleanup := func() {
		tables := []string{
			"user_role", "role_menu", "tenant_application",
			"application_client_secret", "application_client", "menu",
			"role", "user_identity", "user_login_log",
			"department_user", "department", "tenant_user", "person",
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
		// 用 Errorf 而非 Fatalf：一次跑完给出全部数量偏差，避免逐个试错。
		if n != want {
			t.Errorf("table %s: want %d rows, got %d", tbl, want, n)
		}
	}
	assertCount("tenant", 1)
	assertCount("application", 2)
	assertCount("role", 2)
	assertCount("menu", 14)
	assertCount("person", 1)
	assertCount("tenant_user", 1)
	assertCount("application_client", 2)
	assertCount("user_role", 2)
	assertCount("role_menu", 14)
	assertCount("tenant_application", 2)
	assertCount("department", 1)
	assertCount("department_user", 1)

	// 验证管理员用户归属
	var u model.UserEntity
	if err := db.Where("owner_type = ?", model.OwnerTypeOwner).First(&u).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}
	if u.TenantID == "" || u.PersonID == "" {
		t.Fatalf("admin user missing tenant/person linkage: %+v", u)
	}
	t.Logf("admin user id=%s tenant=%s person=%s", u.ID, u.TenantID, u.PersonID)

	// 验证管理员从属顶级部门（primary 行政主部门）
	var rootDept model.DepartmentEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", u.TenantID, "").First(&rootDept).Error; err != nil {
		t.Fatalf("root department not found: %v", err)
	}
	var ou model.DepartmentUserEntity
	if err := db.Where("tenant_id = ? AND user_id = ? AND department_id = ? AND relation_type = ?",
		u.TenantID, u.ID, rootDept.ID, model.DeptUserRelationPrimary).First(&ou).Error; err != nil {
		t.Fatalf("admin department relation not found: %v", err)
	}
	t.Logf("admin dept relation: dept=%s relation=%s", ou.DepartmentID, ou.RelationType)

	// AC-6：PG 侧结构断言——AutoMigrate 实际产出的列类型必须与模型声明一致。
	// SQLite 会忽略 varchar 长度与 jsonb，只有真实 PG 才能验证「16 列确实是 varchar(16)」与
	// 「JSON 列确实是 jsonb」，以及「已下线列/表确实没有被重新创建」。
	type columnRow struct {
		TableName  string
		ColumnName string
		DataType   string
		CharMax    *int64
	}
	columnOf := func(table, column string) (columnRow, bool) {
		t.Helper()
		var row columnRow
		err := db.Raw(`SELECT table_name, column_name, data_type, character_maximum_length AS char_max
			FROM information_schema.columns WHERE table_schema = current_schema()
			  AND table_name = ? AND column_name = ?`, table, column).Scan(&row).Error
		if err != nil {
			t.Fatalf("query column %s.%s: %v", table, column, err)
		}
		return row, row.ColumnName != ""
	}

	// 16 个布尔语义列：抽查各表代表列，必须是 varchar(16)（具名枚举列统一宽度）
	for _, col := range []struct{ table, column string }{
		{"person", "status"},
		{"person", "password_status"},
		{"tenant_user", "status"},
		{"tenant_user", "owner_type"},
		{"domain", "verification_status"},
		{"application", "allow_person_create_tenant"},
		{"application", "allow_join_by_invite"},
		{"application_client", "require_pkce"},
		{"application_client", "require_auth_time"},
		{"menu", "hidden"},
		{"menu", "external_link"},
		{"menu", "keep_alive"},
		{"connector", "allow_auto_create_user"},
		{"connector", "allow_account_link"},
		{"connector", "sync_profile"},
		{"connector", "enable_token_storage"},
	} {
		row, ok := columnOf(col.table, col.column)
		if !ok {
			t.Errorf("列 %s.%s 不存在（P3 改名/枚举化未生效）", col.table, col.column)
			continue
		}
		if row.DataType != "character varying" || row.CharMax == nil || *row.CharMax != 16 {
			t.Errorf("列 %s.%s 类型 = %s(%v)，want varchar(16)", col.table, col.column, row.DataType, row.CharMax)
		}
	}

	// JSON 列：必须是 jsonb（serializer:json + 具名载具类型），抽查覆盖 3 类载具
	for _, col := range []struct{ table, column string }{
		{"application", "role_template"},
		{"application_client", "redirect_uris"},
		{"connector", "config"},
		{"refresh_token", "scopes"},
		{"user_identity", "detail"},
	} {
		row, ok := columnOf(col.table, col.column)
		if !ok {
			t.Errorf("JSON 列 %s.%s 不存在", col.table, col.column)
			continue
		}
		if row.DataType != "jsonb" && row.DataType != "json" {
			t.Errorf("列 %s.%s 类型 = %s，want jsonb/json", col.table, col.column, row.DataType)
		}
	}

	// 已下线列必须不存在（AutoMigrate 只增不删：全新库更不该有）
	for _, col := range []struct{ table, column string }{
		{"tenant_application", "config"},
		{"tenant_application", "granted_scope"},
		{"person", "profile"},
		{"person", "custom_data"},
		{"person", "must_change_password"},
		{"person", "is_suspended"},
		{"tenant_user", "profile"},
		{"tenant_user", "custom_data"},
		{"tenant_user", "is_owner"},
		{"tenant_user", "is_suspended"},
		{"api_key", "scope"},
		{"api_key", "last_used_at"},
		{"user_identity", "last_used_at"},
		{"domain", "is_verified"},
		{"domain", "verified_at"},
	} {
		if row, ok := columnOf(col.table, col.column); ok {
			t.Errorf("已下线列 %s.%s 仍存在（type=%s）", col.table, col.column, row.DataType)
		}
	}

	// 已下线表必须不存在
	for _, tbl := range []string{"log", "system", "scope", "resource", "role_scope"} {
		var n int64
		if err := db.Raw(`SELECT count(*) FROM information_schema.tables
			WHERE table_schema = current_schema() AND table_name = ?`, tbl).Scan(&n).Error; err != nil {
			t.Fatalf("query table %s: %v", tbl, err)
		}
		if n != 0 {
			t.Errorf("已下线表 %s 仍存在", tbl)
		}
	}

	// AC-6 值域断言（校验存量值，不只是列类型）：只断言 DDL 挡不住「不删库直接升级」——
	// AutoMigrate 不改既有列，手工 ALTER ... USING bool::varchar 会把存量值留成文本
	// 'true'/'false'，它们都不是合法枚举值：判 active/suspended、enable/disable 会全部落到
	// 默认分支，其中 require_pkce='true' 还会 fail-open 让 PKCE 静默失守。故逐列校验存量值。
	allowedValues := func(vs ...string) []string { return vs }
	for _, col := range []struct {
		table   string
		column  string
		allowed []string
	}{
		{"person", "status", allowedValues(string(model.PersonStatusActive), string(model.PersonStatusSuspended))},
		{"person", "password_status", allowedValues(string(model.PasswordStatusNormal), string(model.PasswordStatusMustChange))},
		{"tenant_user", "status", allowedValues(string(model.UserStatusActive), string(model.UserStatusSuspended))},
		{"tenant_user", "owner_type", allowedValues(string(model.OwnerTypeOwner), string(model.OwnerTypeNormal))},
		{"domain", "verification_status", allowedValues(string(model.DomainVerificationUnverified), string(model.DomainVerificationVerified))},
		{"application", "allow_person_create_tenant", allowedValues(string(model.AppPersonCreateTenantPolicyEnable), string(model.AppPersonCreateTenantPolicyDisable))},
		{"application", "allow_join_by_invite", allowedValues(string(model.AppJoinByInvitePolicyEnable), string(model.AppJoinByInvitePolicyDisable))},
		{"application_client", "require_pkce", allowedValues(string(model.ClientPKCEPolicyEnable), string(model.ClientPKCEPolicyDisable))},
		{"application_client", "require_auth_time", allowedValues(string(model.ClientAuthTimeClaimPolicyEnable), string(model.ClientAuthTimeClaimPolicyDisable))},
		{"menu", "hidden", allowedValues(string(model.MenuHiddenFlagEnable), string(model.MenuHiddenFlagDisable))},
		{"menu", "external_link", allowedValues(string(model.MenuExternalLinkFlagEnable), string(model.MenuExternalLinkFlagDisable))},
		{"menu", "keep_alive", allowedValues(string(model.MenuKeepAliveFlagEnable), string(model.MenuKeepAliveFlagDisable))},
		{"connector", "allow_auto_create_user", allowedValues(string(model.ConnectorAutoCreateUserFlagEnable), string(model.ConnectorAutoCreateUserFlagDisable))},
		{"connector", "allow_account_link", allowedValues(string(model.ConnectorAccountLinkFlagEnable), string(model.ConnectorAccountLinkFlagDisable))},
		{"connector", "sync_profile", allowedValues(string(model.ConnectorSyncProfileFlagEnable), string(model.ConnectorSyncProfileFlagDisable))},
		{"connector", "enable_token_storage", allowedValues(string(model.ConnectorTokenStorageFlagEnable), string(model.ConnectorTokenStorageFlagDisable))},
	} {
		var illegal int64
		if err := db.Table(col.table).
			Where(col.column+" IS NULL OR "+col.column+" NOT IN ?", col.allowed).
			Count(&illegal).Error; err != nil {
			t.Fatalf("count illegal %s.%s: %v", col.table, col.column, err)
		}
		if illegal != 0 {
			t.Errorf("列 %s.%s 有 %d 行非法枚举值（合法值 %v；boolean→varchar 脏值会留成 'true'/'false'）",
				col.table, col.column, illegal, col.allowed)
		}
	}

	// 值域内的错值抓不住安全不变式：内置客户端的 require_pkce 即使写成合法的 'disable' 也通过了上面的断言，
	// 但 PKCE 未强制意味着授权码可被截获重放，故显式钉住种子必须写 enable。
	for _, code := range []string{model.SeedBuiltinClientPlatformAdminWeb, model.SeedBuiltinClientTenantAdminWeb} {
		var policy string
		if err := db.Raw(`SELECT require_pkce FROM application_client WHERE code = ?`, code).Scan(&policy).Error; err != nil {
			t.Fatalf("query require_pkce of %s: %v", code, err)
		}
		if policy != string(model.ClientPKCEPolicyEnable) {
			t.Errorf("内置客户端 %s 的 require_pkce = %q，want %q", code, policy, model.ClientPKCEPolicyEnable)
		}
	}

	cleanup()
	t.Log("PG AutoMigrate + Seed idempotency check passed")
}
