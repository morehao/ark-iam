package seed_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// TestAutoMigrateNoDataWrite 启动期只建表、**不写任何数据**——这是本次改造最核心的不变式。
//
// 背景：改造前启动期会执行种子写入（租户/管理员/菜单/内置客户端等 43 行）。现在这些写入
// 全部移到初始化页面触发的一次性引导（pkg/seed.Bootstrap），启动期只剩 AutoMigrate。
//
// 为什么必须有一条测试守它：这是"数据由运维决定"的前提。只要启动期还偷偷写一行，
// 运维的删除/改名就会被下次启动撤销，"内置数据归属运维"的承诺立刻失效，
// 而且失效方式是静默的（只体现在重启后数据变了）。所以这里逐个表断言行数为 0，
// 而不是只断言"没有 tenant 行"——写一行别的表同样是违规。
func TestAutoMigrateNoDataWrite(t *testing.T) {
	db, err := openRawSQLite(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 覆盖 L1 会写入的全部表（与黄金基线一致）。
	for _, table := range []string{
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
	} {
		var n int64
		if err := db.Table(table).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Errorf("AutoMigrate 后表 %s 有 %d 行数据，want 0（启动期只建表、不写数据）", table, n)
		}
	}

	// 表结构本身必须已就绪：/install/status 的 schemaReady 判定依据 Model() 能否查到表，
	// 若 AutoMigrate 没能建表，运维看到的是"后端没建表"而不是"没初始化"。
	var tenants []model.TenantEntity
	if err := db.Model(&model.TenantEntity{}).Find(&tenants).Error; err != nil {
		t.Fatalf("AutoMigrate 未建出 tenant 表（schemaReady 会误报 false）: %v", err)
	}
}

// seedTables 是 L1 引导会写入的全部表（与黄金基线一致）。
var seedTables = []string{
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

// TestStartupOnInitializedDatabaseWritesNothing 已初始化库上再走一遍完整启动路径，
// **任何一行数据（含 `updated_at`）都不得变化**。
//
// 与 TestAutoMigrateNoDataWrite 的区别：那条守的是"空库启动不写数据"，这条守的是
// **"已初始化库重启不写数据"**——两者不是同一个断言，后者才是生产上每次发版/每次重启都会发生的路径。
// 少了这条，一个"启动时顺手把内置数据收敛回默认值"的回归（旧 reconcile 语义正是如此）
// 会在所有新库用例里完全看不出来：新库本来就该是默认值。
//
// 快照取**整行内容**而不只是行数：按行数对比抓不住"行数没变但字段被回写"——而后者正是旧机制
// 干的事（`tenant.name` 被收敛回 `平台运营中心`、内置客户端回调被改回默认地址等）。
func TestStartupOnInitializedDatabaseWritesNothing(t *testing.T) {
	db, err := openRawSQLite(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 先完成一次性引导，再做"运维改动"：把若干字段改成人肉值，随后重启不得把它改回去。
	if _, status, err := seed.Bootstrap(context.Background(), db, testDefinition(t)); err != nil {
		t.Fatalf("bootstrap: %v", err)
	} else if status != seed.StatusCreated {
		t.Fatalf("首次引导 status = %q, want %q", status, seed.StatusCreated)
	}
	if err := db.Model(&model.TenantEntity{}).Where("code = ?", model.SeedPlatformTenantCode).
		Update("name", "运维改过的平台名").Error; err != nil {
		t.Fatalf("operator rename: %v", err)
	}

	before := snapshotTables(t, db)

	// 模拟一次完整启动：AutoMigrate → 守卫探针（IsInitialized）→ 结构探测（SchemaReady）。
	// 这三者是启动期与每个请求都会走到的东西，任何一个偷偷写入都必须被这条用例抓住。
	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate (2nd): %v", err)
	}
	initialized, err := seed.IsInitialized(context.Background(), db)
	if err != nil {
		t.Fatalf("IsInitialized: %v", err)
	}
	if !initialized {
		t.Fatal("引导后 IsInitialized = false（自锁判定失效会让守卫永远拦着业务端点）")
	}
	if ready, err := seed.SchemaReady(context.Background(), db); err != nil {
		t.Fatalf("SchemaReady: %v", err)
	} else if !ready {
		t.Fatal("引导后 SchemaReady = false")
	}
	// 再调一次 Bootstrap：真实调用方（/install/initialize）在已初始化库上就是这个调用，
	// 它必须走 AlreadyInitialized 分支且零写入。
	if rep, status, err := seed.Bootstrap(context.Background(), db, testDefinition(t)); err != nil {
		t.Fatalf("bootstrap (2nd): %v", err)
	} else {
		if status != seed.StatusAlreadyInitialized {
			t.Errorf("二次引导 status = %q, want %q", status, seed.StatusAlreadyInitialized)
		}
		if len(rep.Changes) != 0 {
			t.Errorf("二次引导报告了 %d 条变更，want 0：%+v", len(rep.Changes), rep.Changes)
		}
	}

	after := snapshotTables(t, db)
	for _, table := range seedTables {
		if before[table] != after[table] {
			t.Errorf("重启后表 %s 内容发生变化（启动期/二次引导仍在写数据）\n启动前: %s\n启动后: %s",
				table, before[table], after[table])
		}
	}

	// 运维改动必须原样保留（这是"内置数据归运维"的直接证据）。
	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	if tenant.Name != "运维改过的平台名" {
		t.Errorf("租户名 = %q，重启后被改回（种子仍在回写运维改动）", tenant.Name)
	}
}

// snapshotTables 把每张表的全部行内容拼成一个字符串，用于"整库逐字节未变"断言。
//
// 之所以不用 count(*) + max(updated_at)：部分关联表（role_menu/user_role/department_user）
// 没有业务时间列，且"行数不变但字段被回写"恰恰是本次要防的回归；逐行逐列取内容最直接。
func snapshotTables(t *testing.T, db *gorm.DB) map[string]string {
	t.Helper()
	out := make(map[string]string, len(seedTables))
	for _, table := range seedTables {
		var rows []map[string]any
		// Order("rowid") 在 sqlite 上稳定；PG 无 rowid，但同一库两次读取顺序一致，
		// 且下面按格式化后的字符串排序，顺序差异不会造成假阳性。
		if err := db.Table(table).Find(&rows).Error; err != nil {
			t.Fatalf("snapshot %s: %v", table, err)
		}
		parts := make([]string, 0, len(rows))
		for _, row := range rows {
			// 必须用 json.Marshal 而不是 fmt.Sprint：后者对 JSON 列/具名载具类型
			// 会打印出**内存地址**（实测 `role_template:0x4c748653cf0`），
			// 两次读取地址不同会立刻造成假阳性，把"数据没变"误报成"被回写"。
			// json 序列化取的是值本身、且 map 键有序，是可重复的规范形态。
			b, err := json.Marshal(row)
			if err != nil {
				t.Fatalf("marshal %s row: %v", table, err)
			}
			parts = append(parts, string(b))
		}
		sort.Strings(parts)
		out[table] = fmt.Sprintf("rows=%d %s", len(parts), strings.Join(parts, "\n"))
	}
	return out
}
