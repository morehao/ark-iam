package dbclient

import (
	"reflect"
	"strings"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
)

// tenantScopeSkipTables 与 model 层实体表结构的双向交叉守卫（AGENTS.md
// 「上下文传递与租户作用域」第 3 条）：跳过表清单的唯一判定口径是
// 「该实体的 Go 字段 gorm tag 里有没有 column:tenant_id」。
//
// 两个方向都必须红：
//  1. 无 tenant_id 列却没登记 → 带租户上下文的查询会注入 tenant_id 过滤，
//     报表不存在该列（no such column / Unknown column），新增全局表被漏登记；
//  2. 有 tenant_id 列却在跳过表里 → 该表的租户过滤被整体关闭，
//     查询静默返回跨租户数据，属"静默的隔离失效"，比方向 1 更危险。
//
// 守卫不修正任何一侧：不一致即失败，由人决定是补登记还是移出清单。

// tableNameOf 取实体表名（优先 TableName()，与 pkg/model 测试口径一致）。
func tableNameOf(entity any) string {
	if namer, ok := entity.(interface{ TableName() string }); ok {
		return namer.TableName()
	}
	return reflect.TypeOf(entity).String()
}

// gormTagValue 从 gorm tag 中取指定 key 的值（如 column:tenant_id → tenant_id）。
func gormTagValue(tag, key string) string {
	for _, part := range strings.Split(tag, ";") {
		if strings.HasPrefix(part, key+":") {
			return strings.TrimPrefix(part, key+":")
		}
	}
	return ""
}

// entityDeclaresTenantIDColumn 判定实体是否声明 tenant_id 列，递归展开内嵌结构体，
// 以 gorm tag 的 column:tenant_id 为唯一依据（与 tenantScopeSkipTables 的注释口径一致）。
func entityDeclaresTenantIDColumn(entityType reflect.Type) bool {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	if entityType.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		gormTag := field.Tag.Get("gorm")
		if field.Anonymous && gormTagValue(gormTag, "column") == "" {
			if entityDeclaresTenantIDColumn(field.Type) {
				return true
			}
			continue
		}
		if gormTagValue(gormTag, "column") == "tenant_id" {
			return true
		}
	}
	return false
}

// TestTenantScopeSkipTablesRegistryMatchesEntities 双向比对 AllEntities() 的
// 「是否声明 tenant_id 列」与 tenantScopeSkipTables：
// 无列未登记失败；有列却登记（租户隔离被整体关闭）同样失败。
func TestTenantScopeSkipTablesRegistryMatchesEntities(t *testing.T) {
	skipped := make(map[string]struct{}, len(tenantScopeSkipTables))
	for _, table := range tenantScopeSkipTables {
		skipped[table] = struct{}{}
	}
	if len(skipped) != len(tenantScopeSkipTables) {
		t.Errorf("tenantScopeSkipTables 存在重复登记项：%v", tenantScopeSkipTables)
	}

	checked := make(map[string]struct{}, len(model.AllEntities()))
	for _, entity := range model.AllEntities() {
		tableName := tableNameOf(entity)
		if _, ok := checked[tableName]; ok {
			t.Errorf("AllEntities() 存在重复表名：%s", tableName)
			continue
		}
		checked[tableName] = struct{}{}

		hasTenantID := entityDeclaresTenantIDColumn(reflect.TypeOf(entity))
		_, inSkipTables := skipped[tableName]

		switch {
		case !hasTenantID && !inSkipTables:
			t.Errorf("新增全局表未登记：%s 未声明 tenant_id 列，但不在 tenantScopeSkipTables 中；"+
				"带租户上下文的查询会注入 tenant_id 过滤并报 no such column，请登记到 pkg/dbclient/gorm.go",
				tableName)
		case hasTenantID && inSkipTables:
			t.Errorf("陈旧登记导致租户隔离被整体关闭：%s 已声明 tenant_id 列，却仍列在 tenantScopeSkipTables 中；"+
				"该表查询不再注入 tenant_id 过滤（静默的隔离失效），请从跳过表移除",
				tableName)
		}
	}

	// 方向 3：登记了并不存在的表（表已下线或拼写错误）——这类陈旧登记同样是隐患，
	// 会让"这张表是全局表"的认知与实际结构长期不一致。
	for _, table := range tenantScopeSkipTables {
		if _, ok := checked[table]; !ok {
			t.Errorf("tenantScopeSkipTables 登记了不存在的表：%s（表已下线或拼写错误）", table)
		}
	}
}
