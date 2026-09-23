package seed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/gcrypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 白盒用例：直接驱动 bootstrapAll（跳过 Bootstrap 的"已初始化即自锁"判定）。
//
// 为什么需要它：L1 的正常路径只有一次运行，因此"第二次运行也安全"这条**兜底**性质
// 无法从外部（Bootstrap）验证——Bootstrap 第二次直接短路返回。但该性质必须保留：
// 它防御的是锁外重入、进程在提交后崩溃又重试、库被手工清空后重跑等异常路径
// （见设计 §4.2 实现约束 3）。把幂等性与"种子身份键认行"钉在这里，
// 比放在外部用例里更准确——它测的正是内部实现，而不是对外状态机。

// bootstrapForTest 打开全新内存 SQLite + AutoMigrate，并返回直接驱动 bootstrapAll 的闭包。
func bootstrapForTest(t *testing.T) (*gorm.DB, func() Report) {
	t.Helper()
	dsn := fmt.Sprintf("file:bootstrap_all_%d?mode=memory&cache=shared", time.Now().UnixNano())
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

	hash, err := gcrypto.GeneratePasswordHash("BootstrapAll1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	def := defaultDefinition()
	def.AdminPasswordHash = hash

	run := func() Report {
		t.Helper()
		rep := Report{}
		txErr := db.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
			if err := lockSeed(tx); err != nil {
				return err
			}
			return bootstrapAll(context.Background(), tx, &rep, def)
		})
		if txErr != nil {
			t.Fatalf("bootstrapAll fail: %v", txErr)
		}
		return rep
	}
	return db, run
}

// TestBootstrapAll_SecondRunCreatesNothing bootstrapAll 连续执行两次：第二次不得新建任何行。
//
// 这是自锁之外的第二道防线（设计 §4.2 实现约束 3）。
func TestBootstrapAll_SecondRunCreatesNothing(t *testing.T) {
	db, run := bootstrapForTest(t)

	first := run()
	if len(first.Changes) == 0 {
		t.Fatal("首次执行必须产生写入")
	}
	counts := map[string]int64{}
	for _, table := range allSeedTables() {
		counts[table] = countTable(t, db, table)
	}

	second := run()
	if len(second.Changes) != 0 {
		t.Errorf("第二次执行不得新建任何行，实际 %+v", second.Changes)
	}
	for table, want := range counts {
		if got := countTable(t, db, table); got != want {
			t.Errorf("第二次执行后表 %s 行数 = %d, want %d", table, got, want)
		}
	}
}

// TestBootstrapAll_KeepsOperatorEdits 运维改过的字段在第二次执行时不得被回写。
//
// 覆盖三类代表字段（展示名 / 结构 / 编码），它们是历史 bug 的高发区：
//   - tenant.name、application.name：控制台改名后不被拉回种子值（create_only）；
//   - menu.code、menu.name、menu.parent_id：菜单行整体归运维（L1 按 seed_key 认行）；
//   - application_client.name、redirect_uris：客户端展示与运行参数归运维。
//
// 同时验证"种子身份键认行"：改了 code 之后第二次执行仍命中同一行（不新建、不认领错行）。
func TestBootstrapAll_KeepsOperatorEdits(t *testing.T) {
	db, run := bootstrapForTest(t)
	run()

	// 控制台改动：租户改名、应用改名、菜单改名改编码改父级、客户端改名改回调
	if err := db.Model(&model.TenantEntity{}).Where("code = ?", model.SeedPlatformTenantCode).
		Update("name", "ACME 平台运营中心").Error; err != nil {
		t.Fatalf("rename tenant: %v", err)
	}
	// 应用同时改名与改编码：seed_key 认行必须仍然命中同一行（改编码不得另建一行，
	// 这是引入 seed_key 之前的历史 bug——按 (app_id, code) 认行会导致菜单/授权分叉）。
	if err := db.Model(&model.ApplicationEntity{}).Where("seed_key = ?", "platform_admin").
		Updates(map[string]any{"name": "ACME 管理台", "code": "acme_admin"}).Error; err != nil {
		t.Fatalf("rename application: %v", err)
	}
	var rootMenu model.MenuEntity
	if err := db.Where("seed_key = ?", "menu").First(&rootMenu).Error; err != nil {
		t.Fatalf("query menu: %v", err)
	}
	if err := db.Model(&model.MenuEntity{}).Where("id = ?", rootMenu.ID).
		Updates(map[string]any{"name": "菜单字典", "code": "menu-dict", "parent_id": ""}).Error; err != nil {
		t.Fatalf("edit menu: %v", err)
	}
	// 注意：JSON 列必须走实体 + Save（serializer:json 才会生效）。
	// 用 Updates(map[string]any{...}) 会绕过 serializer，静默写入 Go 表示法的脏值
	// （这本身就是 AGENTS.md 明令禁止的写法，这里顺带证明"脏值一旦写入就无法解读"）。
	var client model.ApplicationClientEntity
	if err := db.Where("code = ?", model.SeedBuiltinClientPlatformAdminWeb).First(&client).Error; err != nil {
		t.Fatalf("query client: %v", err)
	}
	client.Name = "ACME SSO 客户端"
	client.RedirectURIs = model.RedirectURIList{"https://admin.acme.com/auth/callback"}
	if err := db.Save(&client).Error; err != nil {
		t.Fatalf("edit client: %v", err)
	}

	menuCountBefore := countTable(t, db, model.TableNameMenu)
	if rep := run(); len(rep.Changes) != 0 {
		t.Fatalf("第二次执行不得写入，实际 %+v", rep.Changes)
	}

	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	if tenant.Name != "ACME 平台运营中心" {
		t.Errorf("租户名 = %q, want 运维自定义值（create_only 不得回写）", tenant.Name)
	}
	var apps []model.ApplicationEntity
	if err := db.Where("seed_key = ?", "platform_admin").Find(&apps).Error; err != nil {
		t.Fatalf("query application by seed_key: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("seed_key=platform_admin 命中 %d 行, want 1（改编码不得新建行）", len(apps))
	}
	if apps[0].Name != "ACME 管理台" {
		t.Errorf("应用名 = %q, want 运维自定义值", apps[0].Name)
	}
	if apps[0].Code != "acme_admin" {
		t.Errorf("应用编码 = %q, want 运维自定义值（create_only 不得回写）", apps[0].Code)
	}
	var appTotal int64
	if err := db.Model(&model.ApplicationEntity{}).Count(&appTotal).Error; err != nil {
		t.Fatalf("count applications: %v", err)
	}
	if appTotal != 2 {
		t.Errorf("应用总行数 = %d, want 2（内置两个应用，改编码不得新增）", appTotal)
	}
	// 按 seed_key 命中同一行：改 code 后既没新建，也没把新值拉回
	var menus []model.MenuEntity
	if err := db.Where("seed_key = ?", "menu").Find(&menus).Error; err != nil {
		t.Fatalf("query menu by seed_key: %v", err)
	}
	if len(menus) != 1 {
		t.Fatalf("seed_key=menu 命中 %d 行, want 1（改 code 不得新建行）", len(menus))
	}
	if menus[0].ID != rootMenu.ID {
		t.Errorf("命中的菜单 id = %s, want %s（必须是同一行）", menus[0].ID, rootMenu.ID)
	}
	if menus[0].Code != "menu-dict" || menus[0].Name != "菜单字典" || menus[0].ParentID != "" {
		t.Errorf("菜单字段被回写: code=%q name=%q parent_id=%q", menus[0].Code, menus[0].Name, menus[0].ParentID)
	}
	if got := countTable(t, db, model.TableNameMenu); got != menuCountBefore {
		t.Errorf("菜单行数 = %d, want %d（不得重建被改编码的菜单）", got, menuCountBefore)
	}

	if err := db.Where("code = ?", model.SeedBuiltinClientPlatformAdminWeb).First(&client).Error; err != nil {
		t.Fatalf("query client: %v", err)
	}
	if client.Name != "ACME SSO 客户端" {
		t.Errorf("客户端名 = %q, want 运维自定义值", client.Name)
	}
	if len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != "https://admin.acme.com/auth/callback" {
		t.Errorf("客户端回调地址 = %v, want 运维自定义值", client.RedirectURIs)
	}
}

// TestBootstrapAll_CreatesMissingRowsOnly 库被部分清空（只删掉某几行）后重跑：
// 只补齐缺失行，不动存在的行。这是"手工清库后重跑"这一异常路径的兜底。
func TestBootstrapAll_CreatesMissingRowsOnly(t *testing.T) {
	db, run := bootstrapForTest(t)
	run()

	// 删掉两个菜单（模拟运维误删/手工清表）
	if err := db.Where("seed_key IN ?", []string{"menu", "oauth-client"}).Delete(&model.MenuEntity{}).Error; err != nil {
		t.Fatalf("delete menus: %v", err)
	}
	before := countTable(t, db, model.TableNameMenu)

	rep := run()
	// 软删行不被"复活"（soft delete 后按 seed_key 查不到），因此这两个菜单会被重建为新行
	if got := countTable(t, db, model.TableNameMenu); got != before+2 {
		t.Errorf("重建后菜单行数 = %d, want %d（缺失行应被补齐）", got, before+2)
	}
	// 只报告缺失行的创建，不报告已存在行
	for _, change := range rep.Changes {
		if change.Entity == model.SeedEntityMenu && change.Action != changeActionCreated {
			t.Errorf("菜单变更动作 = %q, want created", change.Action)
		}
	}
}

// allSeedTables 是 L1 会写入的全部表（与黄金基线一致）。
func allSeedTables() []string {
	return []string{
		model.TableNameTenant,
		model.TableNameDepartment,
		model.TableNameApplication,
		model.TableNameRole,
		model.TableNameMenu,
		model.TableNameRoleMenu,
		model.TableNameTenantApplication,
		model.TableNamePerson,
		model.TableNameUser,
		model.TableNameDepartmentUser,
		model.TableNameUserRole,
		model.TableNameApplicationClient,
	}
}

// countTable 统计表内全量行数（含软删行：本用例关心"有没有被写"）。
func countTable(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
