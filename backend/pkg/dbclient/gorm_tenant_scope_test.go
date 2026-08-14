package dbclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openTestDB 打开独立的内存 sqlite，并挂载与 InitMultiDB 相同的 tenant 插件。
func openTestDB(t *testing.T, skipTables []string) (*gorm.DB, *gin.Context) {
	t.Helper()

	dsn := fmt.Sprintf("file:dbclient_scope_%s_%d?mode=memory&cache=shared",
		strings.NewReplacer("/", "_", " ", "_", ":", "_").Replace(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})

	plugin, err := newTenantScopePlugin(skipTables)
	if err != nil {
		t.Fatalf("newTenantScopePlugin: %v", err)
	}
	if err := db.Use(plugin); err != nil {
		t.Fatalf("db.Use(plugin): %v", err)
	}

	if err := db.AutoMigrate(&model.ApplicationEntity{}, &model.AuditLogEntity{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	// 模拟 OIDC 登录后的请求上下文：gin context 中已写入 tenant_id。
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = &http.Request{URL: &url.URL{}, Header: http.Header{}}
	ginCtx.Set(gcontext.KeyTenantID, uint(1))

	return db, ginCtx
}

// TestTenantScopePluginGlobalTables 回归测试：
// tenant 插件在注入 tenant_id 过滤时必须跳过不含 tenant_id 列的全局表
// （application/menu/user_identity/application_client_secret），
// 否则带租户上下文的请求查询这些表会报
// "Unknown column 'xxx.tenant_id' in 'where clause'"（平台端表现为 [100734] 查看应用列表失败）。
func TestTenantScopePluginGlobalTables(t *testing.T) {
	db, tenantCtx := openTestDB(t, tenantScopeSkipTables)

	if err := db.WithContext(context.Background()).Create(&model.ApplicationEntity{
		Code: "test-app",
		Name: "测试应用",
	}).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}

	// 带租户上下文的查询必须成功（skip 列表覆盖全局表）。
	q := db.WithContext(tenantCtx).Model(&model.ApplicationEntity{}).Table(model.TableNameApplication)
	var count int64
	if err := q.Where(model.TableNameApplication + ".deleted_at IS NULL").Count(&count).Error; err != nil {
		t.Fatalf("query application with tenant context should succeed, got err: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 application row, got %d", count)
	}
}

// TestTenantScopePluginGlobalTablesUnskipped 守卫测试：
// 若跳过列表缺失全局表，插件注入 tenant_id 过滤会直接导致查询失败，
// 证明上面的测试确实能捕获该回归。
func TestTenantScopePluginGlobalTablesUnskipped(t *testing.T) {
	db, tenantCtx := openTestDB(t, []string{"person", "tenant"}) // 旧的（错误的）跳过列表

	q := db.WithContext(tenantCtx).Model(&model.ApplicationEntity{}).Table(model.TableNameApplication)
	var count int64
	err := q.Where(model.TableNameApplication + ".deleted_at IS NULL").Count(&count).Error
	if err == nil {
		t.Fatal("expected query to fail when application is not skipped, got nil err")
	}
}

// TestTenantScopePluginTenantTablesStillScoped 健全性检查：
// 跳过列表只豁免全局表，含 tenant_id 列的表仍必须按租户过滤。
func TestTenantScopePluginTenantTablesStillScoped(t *testing.T) {
	db, tenantCtx := openTestDB(t, tenantScopeSkipTables)

	// 插入两条不同租户的审计日志。
	for _, tenantID := range []uint{1, 2} {
		if err := db.WithContext(context.Background()).Create(&model.AuditLogEntity{
			ActorPersonID: 1,
			Action:        "test.action",
			TargetType:    "application",
			Result:        "success",
			TenantID:      tenantID,
			ClientID:      "c",
		}).Error; err != nil {
			t.Fatalf("seed audit log tenant %d: %v", tenantID, err)
		}
	}

	var rows []model.AuditLogEntity
	if err := db.WithContext(tenantCtx).Model(&model.AuditLogEntity{}).Table(model.TableNameAuditLog).
		Where(model.TableNameAuditLog+".deleted_at IS NULL").Find(&rows).Error; err != nil {
		t.Fatalf("query audit_log: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected only tenant 1 audit log, got %d rows", len(rows))
	}
	if rows[0].TenantID != 1 {
		t.Fatalf("expected tenant_id 1, got %d", rows[0].TenantID)
	}
}
