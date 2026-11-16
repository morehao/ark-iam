package dbclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/morehao/golib/dbaccess/dbgorm"
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/postgres"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/model"
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
// 判断标准：表结构不存在 tenant_id 列（见 pkg/model 各实体定义）。
var tenantScopeSkipTables = []string{
	model.TableNamePerson,
	model.TableNameTenant,
	model.TableNameApplication,
	model.TableNameMenu,
	model.TableNameUserIdentity,
	model.TableNameApplicationClientSecret,
}

// componentSkipTables 是「外部组件自建、非 pkg/model 实体」的全局表（无 tenant_id）。
//
// 这些表由第三方组件维护（AutoMigrate 也由组件负责），不在 model.AllEntities() 里，
// 因此与 tenantScopeSkipTables 分开登记；插件运行时二者合并生效，注册守卫测试对它们放行。
// 登记前确认：该表确实无 tenant_id 列、且确为组件自有表。
var componentSkipTables = []string{
	TableNameCoreSetting, // golib gsetting 运行时配置中心表
}

// TableNameCoreSetting 是 golib gsetting 的配置表名（组件自有，见 gsetting/model.go）。
const TableNameCoreSetting = "core_setting"

func InitMultiDB(configs []dbgorm.Config, logConfig *glog.LogConfig) error {
	if len(configs) == 0 {
		return fmt.Errorf("mysql config is empty")
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
		if err := UseTenantScopePlugin(client); err != nil {
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

// UseTenantScopePlugin 为 db 挂载租户隔离插件（服务初始化与测试脚手架共用同一份配置）。
func UseTenantScopePlugin(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("dbclient: db is nil")
	}
	plugin, err := newTenantScopePlugin(skipTables())
	if err != nil {
		return fmt.Errorf("new tenant scope plugin failed: %w", err)
	}
	return db.Use(plugin)
}

// skipTables 返回租户插件应跳过的全部表：model 实体全局表 + 外部组件全局表。
func skipTables() []string {
	out := make([]string, 0, len(tenantScopeSkipTables)+len(componentSkipTables))
	out = append(out, tenantScopeSkipTables...)
	out = append(out, componentSkipTables...)
	return out
}

// NewStore 实例化一个携带租户隔离插件的数据库连接，供「可嵌入 IAM 库」（iam/）
// 以实例方式持有，而不依赖包级全局 dbMap。与 InitMultiDB 复用同一份租户插件配置，
// 保证内嵌与独立部署的隔离语义一致。
//
// 返回的 *gorm.DB 已挂 UseTenantScopePlugin；调用方负责在退出时关闭底层连接。
func NewStore(cfg dbgorm.Config) (*gorm.DB, error) {
	db, err := dbgorm.New(&cfg)
	if err != nil {
		return nil, fmt.Errorf("dbclient: new store failed: %w", err)
	}
	if err := UseTenantScopePlugin(db); err != nil {
		return nil, fmt.Errorf("dbclient: register tenant plugin failed: %w", err)
	}
	return db, nil
}

// SetIamDB 将给定连接注册为全局 iam 库（生产用）。可嵌入 IAM 库（iam/）在实例化时
// 调用它，使仍通过全局 dbclient.IamDB 取连接的共享代码（pkg/sso、pkg/core 默认路径等）
// 与实例使用同一份连接。一进程一 IAM 实例语义下这是安全且必要的桥接。
func SetIamDB(db *gorm.DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	dbMap[dbNameIam] = db
}

// RegisterDBForTest injects a gorm DB for a service name (test-only usage).
// 测试库与生产库挂载同一份租户隔离配置：未声明租户作用域的查询会 fail-closed，
// 因此测试必须像生产一样显式声明作用域（gin 上下文写租户或 gcontext.WithTenantScope）。
// Callers are responsible for closing the underlying connection.
func RegisterDBForTest(service string, db *gorm.DB) {
	if err := UseTenantScopePlugin(db); err != nil {
		panic(fmt.Sprintf("dbclient: register tenant scope plugin for test db failed: %v", err))
	}
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
