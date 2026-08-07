package dbclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/dbgorm"
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/mysql"
	"github.com/morehao/golib/dbaccess/gormplugin"
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
)

func InitMultiDB(configs []dbgorm.Config, logConfig *glog.LogConfig) error {
	if len(configs) == 0 {
		return fmt.Errorf("mysql config is empty")
	}

	var skipTables = []string{
		"person",
		"tenant",
	}
	tenantPlugin, err := gormplugin.New(&gormplugin.ScopeConfig{
		FieldName: "tenant_id",
		ExtractFunc: func(ctx context.Context) (any, bool) {
			if ginCtx, ok := ctx.(*gin.Context); ok {
				return ginCtx.Get(gcontext.KeyTenantID)
			}
			value := ctx.Value(gcontext.KeyTenantID)
			if value == nil {
				return nil, false
			}
			return value, true
		},
		SkipTables: skipTables,
	})
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
	return dbMap[dbName].WithContext(ctx)
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
