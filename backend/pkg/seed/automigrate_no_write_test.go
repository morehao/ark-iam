package seed_test

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
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
