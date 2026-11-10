package internal

import (
	"context"
	"fmt"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/gorm"
)

// ErrTableConflict 表示目标库存在与 IAM 表同名、但结构不符 IAM 特征的表（同库撞名风险）。
type ErrTableConflict struct{ Table string }

func (e *ErrTableConflict) Error() string {
	return fmt.Sprintf("sdk/server: table %q already exists with a non-IAM structure; "+
		"refuse to AutoMigrate onto it (rename/remove the conflicting table, or add it to Config.AllowExistingTables)", e.Table)
}

// RunMigrations 建表（AutoMigrate「只增不删」）。
//
// 同库内嵌形态下，若目标库已存在同名但非 IAM 结构的表，AutoMigrate 会「加列而非报错」
// 造成静默污染。因此先做撞名检测：对每张 IAM 表，若已存在但缺少 IAM 特征列（id /
// created_at），fail-fast —— 除非在 AllowExistingTables 白名单。
func (a *App) RunMigrations(ctx context.Context) error {
	migCtx := dbclient.CrossTenantContext(ctx)
	db := a.dbGetter(migCtx)
	if db == nil {
		return fmt.Errorf("sdk/server: RunMigrations: db getter returned nil")
	}
	if a.cfg.SameDatabase {
		if err := a.checkTableConflicts(db); err != nil {
			return err
		}
	}
	if err := model.AutoMigrateAll(db); err != nil {
		return fmt.Errorf("sdk/server: automigrate failed: %w", err)
	}
	return nil
}

// iamTableNames 返回 IAM 全部数据表名（源自 model.AllEntities 的 TableName）。
func iamTableNames(db *gorm.DB) ([]string, error) {
	names := make([]string, 0, len(model.AllEntities()))
	for _, entity := range model.AllEntities() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(entity); err != nil {
			return nil, err
		}
		names = append(names, stmt.Schema.Table)
	}
	return names, nil
}

// checkTableConflicts 对每张 IAM 表做撞名检测（同库模式）。
func (a *App) checkTableConflicts(db *gorm.DB) error {
	allow := make(map[string]bool, len(a.cfg.AllowExistingTables))
	for _, t := range a.cfg.AllowExistingTables {
		allow[t] = true
	}
	tables, err := iamTableNames(db)
	if err != nil {
		return fmt.Errorf("sdk/server: resolve table names failed: %w", err)
	}
	existing := existingTables(db)
	for _, t := range tables {
		if allow[t] {
			continue
		}
		if !existing[t] {
			continue
		}
		// 已存在：检查是否具 IAM 特征列（id + created_at）。缺任一 → 视为非 IAM 表，冲突。
		if !tableHasColumn(db, t, "id") || !tableHasColumn(db, t, "created_at") {
			return &ErrTableConflict{Table: t}
		}
	}
	return nil
}

// existingTables 返回当前库已存在的表名集合（PG / sqlite 均可）。
func existingTables(db *gorm.DB) map[string]bool {
	out := make(map[string]bool)
	mig := db.Migrator()
	if tables, err := mig.GetTables(); err == nil {
		for _, t := range tables {
			out[t] = true
		}
	}
	return out
}

// tableHasColumn 判断表是否存在指定列（表不存在返回 false）。
func tableHasColumn(db *gorm.DB, table, col string) bool {
	if !db.Migrator().HasTable(table) {
		return false
	}
	types, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		return false
	}
	for _, ct := range types {
		if ct.Name() == col {
			return true
		}
	}
	return false
}
