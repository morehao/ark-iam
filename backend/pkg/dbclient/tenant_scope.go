package dbclient

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormplugin"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

// tenantScopeField 是租户隔离字段名；所有按租户隔离的表都使用该列。
const tenantScopeField = "tenant_id"

// ErrTenantScopeMissing 表示 ctx 未声明租户作用域：查询/更新/删除被 fail-closed 拒绝。
// 调用方可用 errors.Is 判定；也可用 SetMissingTenantScopeMode 在灰度期降级为告警。
var ErrTenantScopeMissing = errors.New("dbclient: tenant scope missing")

// MissingTenantScopeMode 控制"ctx 未声明租户作用域"时的处置方式。
type MissingTenantScopeMode uint32

const (
	// MissingTenantScopeError fail-closed（默认）：不执行 SQL 并返回 ErrTenantScopeMissing。
	// 缺少作用域的失败模式必须是"响亮报错"，而不是静默跨租户读。
	MissingTenantScopeError MissingTenantScopeMode = iota
	// MissingTenantScopeWarn 仅告警并按历史行为放行（灰度期与紧急回退用）。
	MissingTenantScopeWarn
)

var missingTenantScopeMode atomic.Uint32

func init() {
	missingTenantScopeMode.Store(uint32(MissingTenantScopeError))
}

// SetMissingTenantScopeMode 切换未声明作用域的处置方式，返回旧值。
// 仅供灰度回退与测试使用；生产默认缺省为 fail-closed。
func SetMissingTenantScopeMode(mode MissingTenantScopeMode) MissingTenantScopeMode {
	return MissingTenantScopeMode(missingTenantScopeMode.Swap(uint32(mode)))
}

// CurrentMissingTenantScopeMode 返回当前处置方式。
func CurrentMissingTenantScopeMode() MissingTenantScopeMode {
	return MissingTenantScopeMode(missingTenantScopeMode.Load())
}

// tenantScopeResolver 解析本次数据访问的租户作用域，返回 (过滤值, 是否注入, 是否已声明)。
//
// 声明方式有两类，语义等价，类型化作用域优先：
//  1. gcontext.TenantScope（规范声明）：中间件写入请求上下文，跨 http.Handler 边界的
//     协议层与异步任务同样可见；
//  2. gcontext.KeyTenantID 字符串投影（gin Keys）：gin 对 string key 的 Value 查询
//     无需 ContextWithFallback，因此即使某个引擎漏开开关、或旧代码只写了 gin Keys，
//     隔离也不会静默失效。
//
// 两类都不存在时才判定为"未声明"，交由 MissingScope 钩子决定告警或 fail-closed。
func tenantScopeResolver(ctx context.Context) (any, bool, bool) {
	if value, inject, declared := gcontext.TenantScopeFilter(ctx); declared {
		return value, inject, true
	}
	if value, ok := ctx.Value(gcontext.KeyTenantID).(string); ok && value != "" {
		return value, true, true
	}
	return nil, false, false
}

// handleMissingTenantScope 处置"未声明作用域"的数据访问。
func handleMissingTenantScope(db *gorm.DB, tableName string) error {
	if CurrentMissingTenantScopeMode() == MissingTenantScopeWarn {
		glog.Warnf(db.Statement.Context,
			"[dbclient.tenantScope] 缺少租户作用域，已按历史行为放行（无租户过滤）, table:%s", tableName)
		return nil
	}
	return fmt.Errorf("%w: table=%s; 中间件需用 gincontext.SetTenantScope 写入作用域，"+
		"跨租户访问需在 dao 层用 gcontext.WithTenantScope(ctx, gcontext.AllScope()/ExplicitScope(tenantID)) 显式声明",
		ErrTenantScopeMissing, tableName)
}

// newTenantScopePlugin 构造租户隔离插件（同一份配置供服务与测试复用）。
//
// 实现复用 golib gormplugin.ScopePlugin：其条件构造使用 clause.Column 方言化引用标识符
// （PostgreSQL 生成 "table"."tenant_id"），因此不再需要本项目自建插件。
func newTenantScopePlugin(skipTables []string) (gorm.Plugin, error) {
	return gormplugin.New(&gormplugin.ScopeConfig{
		FieldName:    tenantScopeField,
		Resolver:     tenantScopeResolver,
		MissingScope: handleMissingTenantScope,
		SkipTables:   skipTables,
	})
}

// CrossTenantContext 返回"全部租户"作用域的 ctx：用于按主键/唯一键先定位、再判租户的场景
// （如按 API Key 摘要反查归属租户），必须显式声明，禁止靠"没有作用域"来获得跨租户可见性。
func CrossTenantContext(ctx context.Context) context.Context {
	return gcontext.WithTenantScope(ctx, gcontext.AllScope())
}

// ExplicitTenantContext 返回"指定租户"作用域的 ctx：用于当前请求租户之外的目标租户访问。
func ExplicitTenantContext(ctx context.Context, tenantID string) context.Context {
	return gcontext.WithTenantScope(ctx, gcontext.ExplicitScope(tenantID))
}

// CurrentTenantContext 返回"当前租户"作用域的 ctx：用于非请求入口（后台任务、worker）
// 需要显式声明归属租户的场景。
func CurrentTenantContext(ctx context.Context, tenantID string) context.Context {
	return gcontext.WithTenantScope(ctx, gcontext.CurrentScope(tenantID))
}
