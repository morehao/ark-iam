package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupSQLite 打开独立的内存 SQLite，并注册为全局 iam 库，使 service 内部直接
// dao.NewXxxDao() 的调用自动落到测试库，无需任何注入 seam。
// entities 为需要 AutoMigrate 的 model 结构体（如 &model.UserEntity{}）。
func SetupSQLite(t *testing.T, entities ...any) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(entities...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
