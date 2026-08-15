package dbclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

// tenantScopeSkipKey 与 golib gormplugin.SkipKey 保持一致，
// 保证外部若使用 gormplugin.Skip(db) 跳过租户过滤时同样对本插件生效。
const tenantScopeSkipKey = "gorm:condition:skip"

// tenantScopePlugin 是租户数据隔离插件（PostgreSQL 兼容实现）。
//
// golib gormplugin.ScopePlugin 生成 MySQL 反引号限定符（`table`.tenant_id = ?），
// 在 PostgreSQL 上报 syntax error at or near "."（反引号不是 PG 标识符引用符）；
// 本项目主库为 PG，故在 dbclient 内自建等价实现，限定符改用标准双引号
// （PG / SQLite 均兼容；全部表名已确认非 PG 保留字）。
type tenantScopePlugin struct {
	fieldName   string
	skipTables  map[string]struct{}
	extractFunc func(context.Context) (any, bool)
}

var _ gorm.Plugin = (*tenantScopePlugin)(nil)

// newTenantScopePlugin 构造租户过滤插件，同一份配置供业务初始化与测试复用。
// ExtractFunc 从 gin 上下文（或 context）读取 KeyTenantID：无租户上下文时不注入条件。
func newTenantScopePlugin(skipTables []string) (*tenantScopePlugin, error) {
	skip := make(map[string]struct{}, len(skipTables))
	for _, t := range skipTables {
		if normalized := normalizeTableName(t); normalized != "" {
			skip[normalized] = struct{}{}
		}
	}
	return &tenantScopePlugin{
		fieldName: "tenant_id",
		extractFunc: func(ctx context.Context) (any, bool) {
			if ginCtx, ok := ctx.(*gin.Context); ok {
				return ginCtx.Get(gcontext.KeyTenantID)
			}
			value := ctx.Value(gcontext.KeyTenantID)
			if value == nil {
				return nil, false
			}
			return value, true
		},
		skipTables: skip,
	}, nil
}

// Name 返回插件名（与 golib 一致，便于日志排查）。
func (p *tenantScopePlugin) Name() string { return "scope_condition_plugin" }

// Initialize 注册 query/update/delete 三个阶段的条件注入回调。
func (p *tenantScopePlugin) Initialize(db *gorm.DB) error {
	if strings.TrimSpace(p.fieldName) == "" || p.extractFunc == nil {
		return fmt.Errorf("dbclient: FieldName and ExtractFunc are required")
	}
	callbacks := []struct {
		name   string
		typ    string
		before string
		fn     func(*gorm.DB)
	}{
		{"dbclient:tenant_scope:query", "query", "gorm:query", p.addScope},
		{"dbclient:tenant_scope:update", "update", "gorm:update", p.addScope},
		{"dbclient:tenant_scope:delete", "delete", "gorm:delete", p.addScope},
	}
	for _, cb := range callbacks {
		var registerErr error
		switch cb.typ {
		case "query":
			registerErr = db.Callback().Query().Before(cb.before).Register(cb.name, cb.fn)
		case "update":
			registerErr = db.Callback().Update().Before(cb.before).Register(cb.name, cb.fn)
		case "delete":
			registerErr = db.Callback().Delete().Before(cb.before).Register(cb.name, cb.fn)
		}
		if registerErr != nil {
			return fmt.Errorf("register %s callback: %w", cb.name, registerErr)
		}
	}
	return nil
}

func (p *tenantScopePlugin) addScope(db *gorm.DB) {
	if db.Statement == nil || db.Statement.Context == nil {
		return
	}
	if v, ok := db.Get(tenantScopeSkipKey); ok {
		if skip, ok := v.(bool); ok && skip {
			return
		}
	}
	tableName := resolveTableName(db)
	if tableName == "" {
		return
	}
	if p.isSkipped(tableName) {
		return
	}
	value, ok := p.extractFunc(db.Statement.Context)
	if !ok {
		return
	}
	// PostgreSQL / SQLite 使用双引号作为标识符引用符；
	// 表名与字段名均来自代码常量，非用户输入，无注入风险。
	db.Statement.Where(fmt.Sprintf("\"%s\".\"%s\" = ?", tableName, p.fieldName), value)
}

func (p *tenantScopePlugin) isSkipped(tableName string) bool {
	normalized := normalizeTableName(tableName)
	if normalized == "" {
		return false
	}
	_, ok := p.skipTables[normalized]
	return ok
}

// normalizeTableName 归一化表名：去空白/引号/反引号、剥离 schema 前缀、转小写。
func normalizeTableName(tableName string) string {
	tableName = strings.TrimSpace(tableName)
	tableName = strings.Trim(tableName, "`\"")
	if tableName == "" {
		return ""
	}
	fields := strings.Fields(tableName)
	if len(fields) == 0 {
		return ""
	}
	base := strings.Trim(fields[0], "`\"")
	if idx := strings.LastIndex(base, "."); idx >= 0 {
		base = base[idx+1:]
	}
	return strings.ToLower(base)
}

// resolveTableName 获取当前操作的主表名。
func resolveTableName(db *gorm.DB) string {
	if db.Statement.Table != "" {
		return db.Statement.Table
	}
	if db.Statement.Model != nil {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(db.Statement.Model); err != nil {
			return ""
		}
		return stmt.Table
	}
	return ""
}
