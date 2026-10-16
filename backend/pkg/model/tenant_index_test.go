package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTenantIndexDB 建只含 tenant 的内存库，按生产口径跑 AutoMigrate + EnsurePartialUniqueIndexes。
func newTenantIndexDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:tenant_index_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	// 用与生产启动完全同一条路径（AutoMigrateAll = AutoMigrate + 索引 DDL），
	// 避免测试自己拼装索引而在生产口径变化时失去意义。
	require.NoError(t, AutoMigrateAll(db))
	return db
}

// TestTenantCodePartialUniqueIndexAllowsReuseAfterSoftDelete 是 P0-2c（T7）的验收：
// tenant.code 必须是**部分**唯一索引（WHERE deleted_at IS NULL），
// 否则"软删一个租户再用同一 code 建新租户"会被历史行撞键。
func TestTenantCodePartialUniqueIndexAllowsReuseAfterSoftDelete(t *testing.T) {
	db := newTenantIndexDB(t)

	first := TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "t1"}}, Code: "t_acme", Name: "第一代", Status: TenantStatusActive, Type: TenantTypeCustomer}
	require.NoError(t, db.Create(&first).Error)

	// 软删（gorm.DeletedAt 生效）：行还在表里，但不再占用 code。
	require.NoError(t, db.Delete(&TenantEntity{}, "id = ?", "t1").Error)

	second := TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "t2"}}, Code: "t_acme", Name: "第二代", Status: TenantStatusActive, Type: TenantTypeCustomer}
	require.NoError(t, db.Create(&second).Error, "软删后同一 code 必须可以再次插入（部分唯一索引只约束未删除行）")

	// 未删除行之间仍然唯一：同 code 的第三条活行必须被拒绝。
	third := TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "t3"}}, Code: "t_acme", Name: "第三条", Status: TenantStatusActive, Type: TenantTypeCustomer}
	assert.Error(t, db.Create(&third).Error, "未删除行之间的 code 唯一性必须仍然生效")
}

// tenantEntityWithPlainUniqueCode 是 T7 改动前的口径（普通 uniqueIndex）的对照组：
// 它证明上面的测试确实能发现"索引口径退化为普通唯一索引"这一回归（而不是恒真的断言）。
type tenantEntityWithPlainUniqueCode struct {
	gormdao.BaseEntity
	Code string `gorm:"column:code;type:varchar(64);not null;default:'';uniqueIndex"`
}

func (tenantEntityWithPlainUniqueCode) TableName() string { return "tenant_plain_unique_probe" }

// TestTenantCodePlainUniqueIndexBlocksReuse 对照组（自证有效）：
// 同样的"软删 + 复用 code"在普通唯一索引下会失败——正是 T7 要修掉的存量口径。
func TestTenantCodePlainUniqueIndexBlocksReuse(t *testing.T) {
	dsn := fmt.Sprintf("file:tenant_plain_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&tenantEntityWithPlainUniqueCode{}))

	require.NoError(t, db.Create(&tenantEntityWithPlainUniqueCode{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "p1"}}, Code: "t_acme"}).Error)
	require.NoError(t, db.Delete(&tenantEntityWithPlainUniqueCode{}, "id = ?", "p1").Error)

	err = db.Create(&tenantEntityWithPlainUniqueCode{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "p2"}}, Code: "t_acme"}).Error
	require.Error(t, err, "普通唯一索引下软删行仍占用 code——这正是 T7 要消除的行为")
	t.Logf("对照组（普通唯一索引）复用 code 的报错：%v", err)
}
