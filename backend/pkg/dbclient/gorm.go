package dbclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/morehao/golib/dbaccess/dbgorm"
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/postgres"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

var (
	dbMap   = make(map[string]*gorm.DB)
	dbMutex sync.RWMutex
)

const (
	dbNameDemo = "demo"
	dbNameIam  = "iam"

	// ServiceNameIam iam 库服务名，测试中配合 RegisterDBForTest 使用。
	ServiceNameIam = dbNameIam
)

// tenantScopeSkipTables 列出不含 tenant_id 列的全局表。
// tenant 插件会为所有未跳过的表自动注入 tenant_id 过滤条件，而这些表没有该列，
// 注入后查询会报 "Unknown column 'xxx.tenant_id' in 'where clause'"
// （平台端表现为 [100734] 查看应用列表失败等列表类错误）。
// 判断标准：表结构不存在 tenant_id 列（见 pkg/iam/model 各实体定义）。
var tenantScopeSkipTables = []string{
	"person",
	"tenant",
	"application",
	"menu",
	"user_identity",
	"application_client_secret",
}

func InitMultiDB(configs []dbgorm.Config, logConfig *glog.LogConfig) error {
	if len(configs) == 0 {
		return fmt.Errorf("mysql config is empty")
	}

	tenantPlugin, err := newTenantScopePlugin(tenantScopeSkipTables)
	if err != nil {
		return fmt.Errorf("init tenant plugin failed: %v", err)
	}

	var opts []dbgorm.Option
	if logConfig != nil {
		opts = append(opts, dbgorm.WithLogConfig(logConfig))
	}
	opts = append(opts, dbgorm.WithCallerSkip(3))
	for _, cfg := range configs {
		client, err := dbgorm.New(&cfg, opts...)
		if err != nil {
			return fmt.Errorf("init mysql failed: %v", err)
		}
		if err := client.Use(tenantPlugin); err != nil {
			return fmt.Errorf("register tenant plugin failed: %v", err)
		}
		dbMutex.Lock()
		dbMap[cfg.Service] = client
		dbMutex.Unlock()
	}
	return nil
}

func GetDB(ctx context.Context, dbName string) *gorm.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	db := dbMap[dbName]
	if db == nil {
		// 库未注册（如未 InitMultiDB 的测试环境）时返回 nil，调用方自行判空，
		// 避免 nil *gorm.DB 方法调用直接 panic。
		return nil
	}
	return db.WithContext(ctx)
}

func IamDB(ctx context.Context) *gorm.DB {
	return GetDB(ctx, dbNameIam)
}

// RegisterDBForTest injects a gorm DB for a service name (test-only usage).
// Callers are responsible for closing the underlying connection.
func RegisterDBForTest(service string, db *gorm.DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	dbMap[service] = db
}

// ClearDBForTest removes a registered DB (test-only usage).
func ClearDBForTest(service string) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	delete(dbMap, service)
}

func DemoDB(ctx context.Context) *gorm.DB {
	return GetDB(ctx, dbNameDemo)
}
