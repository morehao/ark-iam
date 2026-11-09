package seed_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
	"github.com/morehao/golib/gcrypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// testDefinition 返回只填了口令摘要的定义：其余字段刻意留空，用于覆盖"零值回落内置缺省"。
func testDefinition(t *testing.T) seed.Definition {
	t.Helper()
	hash, err := gcrypto.GeneratePasswordHash("Admin123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return seed.Definition{AdminPasswordHash: hash}
}

// countRows 统计表行数（L1 幂等性与"零写入"断言的通用工具）。
func countRows(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// openRawSQLite 打开独立内存 SQLite 但**不**建表：用于覆盖"表未就绪"的边界。
func openRawSQLite(t *testing.T) (*gorm.DB, error) {
	t.Helper()
	dsn := fmt.Sprintf("file:seed_raw_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db, nil
}

// TestBootstrap_RequiresPasswordHash 口令摘要缺失属调用错误：不写任何数据，也不落成"已初始化"。
func TestBootstrap_RequiresPasswordHash(t *testing.T) {
	db := setupDB(t)
	rep, status, err := seed.Bootstrap(context.Background(), db, seed.Definition{})
	if err != seed.ErrAdminPasswordHashRequired {
		t.Fatalf("err = %v, want ErrAdminPasswordHashRequired", err)
	}
	if status != "" {
		t.Errorf("status = %q, want 空（失败不得伪装成状态）", status)
	}
	if len(rep.Changes) != 0 {
		t.Errorf("Changes = %v, want 空", rep.Changes)
	}
	if n := countRows(t, db, model.TableNameTenant); n != 0 {
		t.Errorf("tenant 行数 = %d, want 0（失败必须零写入）", n)
	}
}

// TestBootstrap_ZeroDefinitionFallsBack 零值定义必须产出与改造前启动播种一致的内置数据。
func TestBootstrap_ZeroDefinitionFallsBack(t *testing.T) {
	db := setupDB(t)
	rep, status, err := seed.Bootstrap(context.Background(), db, testDefinition(t))
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if status != seed.StatusCreated {
		t.Fatalf("status = %q, want %q", status, seed.StatusCreated)
	}
	if rep.TenantID == "" {
		t.Error("Report.TenantID 为空：审计与响应需要它定位目标租户")
	}

	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	if tenant.Name != "平台运营中心" {
		t.Errorf("tenant.Name = %q, want 平台运营中心", tenant.Name)
	}
	if rep.TenantID != tenant.ID {
		t.Errorf("Report.TenantID = %q, want %q", rep.TenantID, tenant.ID)
	}

	// 根部门名派生自租户名
	var dept model.DepartmentEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&dept).Error; err != nil {
		t.Fatalf("query root department: %v", err)
	}
	if dept.Name != tenant.Name {
		t.Errorf("根部门名 = %q, want 与租户同名 %q", dept.Name, tenant.Name)
	}

	var person model.PersonEntity
	if err := db.Where("username = ?", "admin").First(&person).Error; err != nil {
		t.Fatalf("query person: %v", err)
	}
	if person.PrimaryEmail == nil || *person.PrimaryEmail != "admin@example.com" {
		t.Errorf("PrimaryEmail = %v", person.PrimaryEmail)
	}
	if person.PrimaryPhone == nil || *person.PrimaryPhone != "13800000000" {
		t.Errorf("PrimaryPhone = %v", person.PrimaryPhone)
	}
	if person.Name != "系统管理员" {
		t.Errorf("person.Name = %q, want 系统管理员", person.Name)
	}
	if person.PasswordStatus != model.PasswordStatusNormal {
		t.Errorf("PasswordStatus = %q, want %q", person.PasswordStatus, model.PasswordStatusNormal)
	}

	var user model.UserEntity
	if err := db.Where("tenant_id = ? AND person_id = ?", tenant.ID, person.ID).First(&user).Error; err != nil {
		t.Fatalf("query user: %v", err)
	}
	if user.Name != "系统管理员" {
		t.Errorf("user.Name = %q, want 系统管理员", user.Name)
	}

	// 回调地址：零值定义必须落回 dev 缺省，且 bc-logout 由 issuer 派生
	var client model.ApplicationClientEntity
	if err := db.Where("code = ?", model.SeedBuiltinClientPlatformAdminWeb).First(&client).Error; err != nil {
		t.Fatalf("query builtin client: %v", err)
	}
	if len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != "http://localhost:4001/auth/callback" {
		t.Errorf("redirect_uris = %v", client.RedirectURIs)
	}
	if client.BackChannelLogoutURI != "http://localhost:8100/oidc/bc-logout/platform" {
		t.Errorf("back_channel_logout_uri = %q", client.BackChannelLogoutURI)
	}
}

// TestBootstrap_UsesDefinitionInputs 页面填入的身份信息与部署拓扑必须逐字段落库。
func TestBootstrap_UsesDefinitionInputs(t *testing.T) {
	db := setupDB(t)
	hash, err := gcrypto.GeneratePasswordHash("Acme#Pass1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	def := seed.Definition{
		TenantName:          "Acme 平台运营中心",
		AdminUsername:       "acme-root",
		AdminName:           "张运维",
		AdminEmail:          "ops@acme.example",
		AdminPhone:          "13900000009",
		AdminPasswordHash:   hash,
		AdminPasswordStatus: model.PasswordStatusMustChange,
		Issuer:              "https://sso.acme.example/oidc",
		Consoles: seed.ConsolesConfig{
			PlatformAdminWeb: seed.ConsoleRedirects{
				RedirectURIs:           []string{"https://admin.acme.example/auth/callback"},
				PostLogoutRedirectURIs: []string{"https://admin.acme.example/login"},
			},
			TenantAdminWeb: seed.ConsoleRedirects{
				RedirectURIs:           []string{"https://tenant.acme.example/auth/callback"},
				PostLogoutRedirectURIs: []string{"https://tenant.acme.example/login"},
				BackChannelLogoutURI:   "https://tenant-rcv.acme.example/bc",
			},
		},
	}
	if _, status, err := seed.Bootstrap(context.Background(), db, def); err != nil {
		t.Fatalf("bootstrap: %v", err)
	} else if status != seed.StatusCreated {
		t.Fatalf("status = %q", status)
	}

	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	if tenant.Name != def.TenantName {
		t.Errorf("tenant.Name = %q, want %q", tenant.Name, def.TenantName)
	}
	var dept model.DepartmentEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&dept).Error; err != nil {
		t.Fatalf("query root department: %v", err)
	}
	if dept.Name != def.TenantName {
		t.Errorf("根部门名 = %q, want 派生自租户名 %q", dept.Name, def.TenantName)
	}

	var person model.PersonEntity
	if err := db.Where("username = ?", def.AdminUsername).First(&person).Error; err != nil {
		t.Fatalf("query person by custom username: %v", err)
	}
	if person.PrimaryEmail == nil || *person.PrimaryEmail != def.AdminEmail {
		t.Errorf("PrimaryEmail = %v, want %q", person.PrimaryEmail, def.AdminEmail)
	}
	if person.PrimaryPhone == nil || *person.PrimaryPhone != def.AdminPhone {
		t.Errorf("PrimaryPhone = %v, want %q", person.PrimaryPhone, def.AdminPhone)
	}
	if person.Name != def.AdminName {
		t.Errorf("person.Name = %q, want %q", person.Name, def.AdminName)
	}
	if person.PasswordStatus != model.PasswordStatusMustChange {
		t.Errorf("PasswordStatus = %q, want %q", person.PasswordStatus, model.PasswordStatusMustChange)
	}
	// 口令摘要必须是调用方给的这一份（页面指定的口令，而非任何内置默认口令）
	if err := gcrypto.ComparePasswordHash(person.PasswordEncrypted, "Acme#Pass1"); err != nil {
		t.Errorf("页面指定的口令校验失败: %v", err)
	}
	if err := gcrypto.ComparePasswordHash(person.PasswordEncrypted, "admin123"); err == nil {
		t.Error("内置历史默认口令 admin123 竟然可以登录：默认口令未被彻底移除")
	}
	// 默认用户名不得被建出来
	if n := countRows(t, db, model.TableNamePerson); n != 1 {
		t.Errorf("person 行数 = %d, want 1（不得同时建出默认 admin）", n)
	}

	var user model.UserEntity
	if err := db.Where("tenant_id = ? AND person_id = ?", tenant.ID, person.ID).First(&user).Error; err != nil {
		t.Fatalf("query user: %v", err)
	}
	if user.Name != def.AdminName {
		t.Errorf("user.Name = %q, want %q", user.Name, def.AdminName)
	}

	// 回调地址：配置写什么就是什么（零派生、零拼接）
	clients := map[string]seed.ConsoleRedirects{
		model.SeedBuiltinClientPlatformAdminWeb: def.Consoles.PlatformAdminWeb,
		model.SeedBuiltinClientTenantAdminWeb:   def.Consoles.TenantAdminWeb,
	}
	for code, want := range clients {
		var client model.ApplicationClientEntity
		if err := db.Where("code = ?", code).First(&client).Error; err != nil {
			t.Fatalf("query client %s: %v", code, err)
		}
		if len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != want.RedirectURIs[0] {
			t.Errorf("%s redirect_uris = %v, want %v", code, client.RedirectURIs, want.RedirectURIs)
		}
		if len(client.PostLogoutRedirectURIs) != 1 || client.PostLogoutRedirectURIs[0] != want.PostLogoutRedirectURIs[0] {
			t.Errorf("%s post_logout_redirect_uris = %v, want %v", code, client.PostLogoutRedirectURIs, want.PostLogoutRedirectURIs)
		}
	}
	var platformClient model.ApplicationClientEntity
	if err := db.Where("code = ?", model.SeedBuiltinClientPlatformAdminWeb).First(&platformClient).Error; err != nil {
		t.Fatalf("query platform client: %v", err)
	}
	if platformClient.BackChannelLogoutURI != "https://sso.acme.example/oidc/bc-logout/platform" {
		t.Errorf("未覆盖的 bc-logout 应由自定义 issuer 派生, got %q", platformClient.BackChannelLogoutURI)
	}
	var tenantClient model.ApplicationClientEntity
	if err := db.Where("code = ?", model.SeedBuiltinClientTenantAdminWeb).First(&tenantClient).Error; err != nil {
		t.Fatalf("query tenant client: %v", err)
	}
	if tenantClient.BackChannelLogoutURI != "https://tenant-rcv.acme.example/bc" {
		t.Errorf("显式覆盖的 bc-logout 未生效, got %q", tenantClient.BackChannelLogoutURI)
	}
}

// TestBootstrap_SecondCallIsNoop 已初始化后再调用：状态为 already_initialized 且全表零写入。
//
// 自锁的判定依据是库内事实（平台租户行），因此重启、换副本、删审计日志都不影响结论。
func TestBootstrap_SecondCallIsNoop(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	def := testDefinition(t)
	if _, status, err := seed.Bootstrap(ctx, db, def); err != nil || status != seed.StatusCreated {
		t.Fatalf("first bootstrap: status=%q err=%v", status, err)
	}

	tables := []string{
		model.TableNameTenant, model.TableNameDepartment, model.TableNameApplication,
		model.TableNameRole, model.TableNameMenu, model.TableNameRoleMenu,
		model.TableNameTenantApplication, model.TableNamePerson, model.TableNameUser,
		model.TableNameDepartmentUser, model.TableNameUserRole, model.TableNameApplicationClient,
	}
	before := make(map[string]int64, len(tables))
	for _, tbl := range tables {
		before[tbl] = countRows(t, db, tbl)
	}

	rep, status, err := seed.Bootstrap(ctx, db, def)
	if err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if status != seed.StatusAlreadyInitialized {
		t.Fatalf("status = %q, want %q", status, seed.StatusAlreadyInitialized)
	}
	if len(rep.Changes) != 0 {
		t.Errorf("Changes = %v, want 空（已初始化不得有任何变更）", rep.Changes)
	}
	for _, tbl := range tables {
		if got := countRows(t, db, tbl); got != before[tbl] {
			t.Errorf("表 %s 行数 %d -> %d：已初始化的库被二次写入", tbl, before[tbl], got)
		}
	}
}

// TestIsInitialized 自锁判定的正反向。
func TestIsInitialized(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	got, err := seed.IsInitialized(ctx, db)
	if err != nil {
		t.Fatalf("IsInitialized: %v", err)
	}
	if got {
		t.Error("空库 IsInitialized = true, want false")
	}

	if _, _, err := seed.Bootstrap(ctx, db, testDefinition(t)); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	got, err = seed.IsInitialized(ctx, db)
	if err != nil {
		t.Fatalf("IsInitialized: %v", err)
	}
	if !got {
		t.Error("Bootstrap 之后 IsInitialized = false, want true")
	}
}

// TestSchemaReady 区分"表未就绪"与"表已就绪但数据未写入"两种未初始化状态。
func TestSchemaReady(t *testing.T) {
	ctx := context.Background()

	rawDB, err := openRawSQLite(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	ready, err := seed.SchemaReady(ctx, rawDB)
	if err != nil {
		t.Fatalf("SchemaReady: %v", err)
	}
	if ready {
		t.Error("未建表时 SchemaReady = true, want false")
	}

	if err := model.AutoMigrateAll(rawDB); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	ready, err = seed.SchemaReady(ctx, rawDB)
	if err != nil {
		t.Fatalf("SchemaReady: %v", err)
	}
	if !ready {
		t.Error("AutoMigrate 之后 SchemaReady = false, want true")
	}
	// 表已就绪但数据未写入：SchemaReady true 且 IsInitialized false
	if initialized, err := seed.IsInitialized(ctx, rawDB); err != nil {
		t.Fatalf("IsInitialized: %v", err)
	} else if initialized {
		t.Error("仅建表不应视为已初始化")
	}
}
