package dbclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newScopeTestDB 建一个独立内存库：挂载与生产相同的租户插件，并写入两个租户的审计日志。
func newScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:scope_enforce_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, cErr := db.DB(); cErr == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, UseTenantScopePlugin(db))
	require.NoError(t, db.AutoMigrate(&model.AuditLogEntity{}))

	// 种子写入跨租户：显式声明「全部租户」作用域（与启动期种子一致）。
	seedCtx := gcontext.WithTenantScope(context.Background(), gcontext.AllScope())
	for _, tenantID := range []string{"t1", "t2"} {
		require.NoError(t, db.WithContext(seedCtx).Create(&model.AuditLogEntity{
			ActorPersonID: "p1",
			Action:        "test.action",
			TargetType:    "application",
			Result:        "success",
			TenantID:      tenantID,
			ClientID:      "c",
		}).Error)
	}
	return db
}

// newGinCtx 构造真实引擎下的 gin 上下文：fallback 对应 engine.ContextWithFallback。
func newGinCtx(t *testing.T, fallback bool, scope gcontext.TenantScope) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.ContextWithFallback = fallback

	var captured *gin.Context
	engine.GET("/probe", func(c *gin.Context) {
		gincontext.SetTenantScope(c, scope)
		captured = c
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/probe", nil)
	require.NoError(t, err)
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured)
	// 请求结束后 gin 会复用上下文，这里复制出仅用于读值的等价上下文，
	// 使断言不依赖请求生命周期（AsyncContext 也是同样思路）。
	return captured.Copy()
}

func queryAuditTenants(t *testing.T, db *gorm.DB, ctx context.Context) ([]string, error) {
	t.Helper()
	var rows []model.AuditLogEntity
	err := db.WithContext(ctx).Model(&model.AuditLogEntity{}).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	tenants := make([]string, 0, len(rows))
	for i := range rows {
		tenants = append(tenants, rows[i].TenantID)
	}
	return tenants, nil
}

// TestTenantScopeTypedCarrierOnGinContext 类型化作用域经 gin 上下文可直接被数据层解析：
// 业务侧只传 *gin.Context（不出现 .Request.Context()），底层不做任何载体断言。
func TestTenantScopeTypedCarrierOnGinContext(t *testing.T) {
	db := newScopeTestDB(t)

	ginCtx := newGinCtx(t, true, gcontext.CurrentScope("t1"))
	tenants, err := queryAuditTenants(t, db, ginCtx)
	require.NoError(t, err)
	require.Equal(t, []string{"t1"}, tenants, "gin ctx 直传必须按当前租户过滤")

	explicitCtx := newGinCtx(t, true, gcontext.ExplicitScope("t2"))
	tenants, err = queryAuditTenants(t, db, explicitCtx)
	require.NoError(t, err)
	require.Equal(t, []string{"t2"}, tenants, "指定租户作用域必须按目标租户过滤")

	allCtx := newGinCtx(t, true, gcontext.AllScope())
	tenants, err = queryAuditTenants(t, db, allCtx)
	require.NoError(t, err)
	require.Len(t, tenants, 2, "全租户作用域不过滤")
}

// TestTenantScopeMissingFailsClosed 未声明作用域必须响亮报错，绝不能静默跨租户读。
func TestTenantScopeMissingFailsClosed(t *testing.T) {
	db := newScopeTestDB(t)

	_, err := queryAuditTenants(t, db, context.Background())
	require.ErrorIs(t, err, ErrTenantScopeMissing)

	var rows []model.AuditLogEntity
	tx := db.WithContext(context.Background()).Model(&model.AuditLogEntity{}).Find(&rows)
	require.ErrorIs(t, tx.Error, ErrTenantScopeMissing)
	require.Empty(t, tx.Statement.SQL.String(), "fail-closed 时不得生成 SQL")
}

// TestTenantScopeMissingWarnModeIsReversible 灰度回退：告警模式放行且可恢复。
func TestTenantScopeMissingWarnModeIsReversible(t *testing.T) {
	db := newScopeTestDB(t)

	previous := SetMissingTenantScopeMode(MissingTenantScopeWarn)
	t.Cleanup(func() { SetMissingTenantScopeMode(previous) })

	tenants, err := queryAuditTenants(t, db, context.Background())
	require.NoError(t, err)
	require.Len(t, tenants, 2, "告警模式保持历史行为（不过滤）")

	require.Equal(t, MissingTenantScopeWarn, CurrentMissingTenantScopeMode())
	SetMissingTenantScopeMode(previous)
	require.Equal(t, MissingTenantScopeError, CurrentMissingTenantScopeMode())
}

// TestTenantScopeGinCtxEqualsRequestContext 是 G2 的验收：
// 同一请求下 ctx 与 ctx.Request.Context() 解析出的租户过滤完全一致（逐字节相同 SQL）。
func TestTenantScopeGinCtxEqualsRequestContext(t *testing.T) {
	db := newScopeTestDB(t)
	ginCtx := newGinCtx(t, true, gcontext.CurrentScope("t1"))

	var viaGin []model.AuditLogEntity
	txGin := db.Session(&gorm.Session{DryRun: true}).WithContext(ginCtx).
		Model(&model.AuditLogEntity{}).Find(&viaGin)
	require.NoError(t, txGin.Error)

	var viaRequest []model.AuditLogEntity
	txRequest := db.Session(&gorm.Session{DryRun: true}).WithContext(ginCtx.Request.Context()).
		Model(&model.AuditLogEntity{}).Find(&viaRequest)
	require.NoError(t, txRequest.Error)

	require.Equal(t, txGin.Statement.SQL.String(), txRequest.Statement.SQL.String())
	require.Equal(t, txGin.Statement.Vars, txRequest.Statement.Vars)
	require.Contains(t, txGin.Statement.SQL.String(), "tenant_id")

	// 真实执行结果同样一致
	tenantsGin, err := queryAuditTenants(t, db, ginCtx)
	require.NoError(t, err)
	tenantsRequest, err := queryAuditTenants(t, db, ginCtx.Request.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"t1"}, tenantsGin)
	require.Equal(t, tenantsGin, tenantsRequest)
}

// TestTenantScopeGinKeyProjectionAlsoDeclares 兼容声明：
// 即使引擎漏开 ContextWithFallback（或旧代码只写 gin Keys），字符串投影仍能驱动隔离，
// 隔离不会因为一个开关漏配而静默失效。
func TestTenantScopeGinKeyProjectionAlsoDeclares(t *testing.T) {
	db := newScopeTestDB(t)

	// 关闭 fallback 的引擎：类型化值在 gin ctx 上不可见，只剩 gin Keys 投影。
	ginCtx := newGinCtx(t, false, gcontext.CurrentScope("t1"))
	_, typedVisible := gcontext.TenantScopeFrom(ginCtx)
	require.False(t, typedVisible, "未开 fallback 时类型化作用域在 gin ctx 上不可见")

	tenants, err := queryAuditTenants(t, db, ginCtx)
	require.NoError(t, err)
	require.Equal(t, []string{"t1"}, tenants, "gin Keys 投影仍须保证按租户过滤")

	// 仅写入 gin Keys（历史写法）同样被认作已声明。
	legacy, _ := gin.CreateTestContext(httptest.NewRecorder())
	legacy.Request = httptest.NewRequest(http.MethodGet, "/probe", nil)
	legacy.Set(gcontext.KeyTenantID, "t2")
	tenants, err = queryAuditTenants(t, db, legacy)
	require.NoError(t, err)
	require.Equal(t, []string{"t2"}, tenants)
}

// TestTenantScopeAsyncContextKeepsScope 异步续跑：AsyncContext 带走作用域、丢掉取消。
func TestTenantScopeAsyncContextKeepsScope(t *testing.T) {
	db := newScopeTestDB(t)
	ginCtx := newGinCtx(t, true, gcontext.CurrentScope("t1"))

	asyncCtx := gincontext.AsyncContext(ginCtx)
	tenants, err := queryAuditTenants(t, db, asyncCtx)
	require.NoError(t, err)
	require.Equal(t, []string{"t1"}, tenants, "异步任务必须沿用请求租户作用域")
	require.Nil(t, asyncCtx.Done(), "异步 context 不得继承请求取消")
}

// TestTenantScopeHelpers 便捷构造器语义。
func TestTenantScopeHelpers(t *testing.T) {
	base := context.Background()

	value, inject, declared := gcontext.TenantScopeFilter(CrossTenantContext(base))
	require.Nil(t, value)
	require.False(t, inject)
	require.True(t, declared)

	value, inject, declared = gcontext.TenantScopeFilter(ExplicitTenantContext(base, "t9"))
	require.Equal(t, "t9", value)
	require.True(t, inject)
	require.True(t, declared)

	value, inject, declared = gcontext.TenantScopeFilter(CurrentTenantContext(base, "t8"))
	require.Equal(t, "t8", value)
	require.True(t, inject)
	require.True(t, declared)
}
