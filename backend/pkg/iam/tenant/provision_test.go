package tenant

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newProvisionTestDB 内存 SQLite 承载权限开通涉及的全部表。
func newProvisionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:iam_provision_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.ApplicationEntity{},
		&model.MenuEntity{},
		&model.RoleEntity{},
		&model.RoleMenuEntity{},
		&model.TenantApplicationEntity{},
		&model.UserRoleEntity{},
	))
	t.Cleanup(func() {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// seedTenantAdminApp 写入 tenant-admin 应用与指定菜单（模拟 pkg/seed 的全局种子数据）。
func seedTenantAdminApp(t *testing.T, db *gorm.DB, menuCodes ...string) *model.ApplicationEntity {
	t.Helper()
	ctx := context.Background()
	app := &model.ApplicationEntity{Code: ProvisionAppCode, Name: "租户自服务", Status: model.AppStatusEnable}
	require.NoError(t, db.WithContext(ctx).Create(app).Error)
	for i, code := range menuCodes {
		require.NoError(t, db.WithContext(ctx).Create(&model.MenuEntity{
			AppID:      app.ID,
			Name:       code,
			Code:       code,
			Path:       "/" + code,
			Sort:       i + 1,
			Type:       model.MenuTypeMenu,
			Visibility: model.MenuVisibilityAdmin,
			Status:     model.MenuStatusEnable,
		}).Error)
	}
	return app
}

func countProvisionRows(t *testing.T, db *gorm.DB, entity any, query string, args ...any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(entity).Where(query, args...).Count(&count).Error)
	return count
}

func TestProvisionTenantAdmin_CreatesSubscriptionRoleMenusAndGrant(t *testing.T) {
	db := newProvisionTestDB(t)
	app := seedTenantAdminApp(t, db, ProvisionMenuCodes...)

	role, err := ProvisionTenantAdmin(context.Background(), db, &ProvisionTenantAdminReq{
		TenantID:    "t1",
		GrantUserID: "u1",
		CreatedBy:   "operator1",
	})
	require.NoError(t, err)
	require.NotNil(t, role)
	require.Equal(t, ProvisionRoleName, role.Name)
	require.Equal(t, app.ID, role.AppID)
	require.Equal(t, string(model.RoleSourceBuiltin), role.Source)
	require.Equal(t, ProvisionAdminType, role.AdminType)
	require.True(t, role.IsBuiltinAdmin())

	// 应用订阅 1 + 角色 1 + 授权 4 + 管理员绑定 1
	require.Equal(t, int64(1), countProvisionRows(t, db, &model.TenantApplicationEntity{}, "tenant_id = ? AND app_id = ?", "t1", app.ID))
	require.Equal(t, int64(1), countProvisionRows(t, db, &model.RoleEntity{}, "tenant_id = ? AND app_id = ? AND source = ?", "t1", app.ID, string(model.RoleSourceBuiltin)))
	require.Equal(t, int64(len(ProvisionMenuCodes)), countProvisionRows(t, db, &model.RoleMenuEntity{}, "tenant_id = ? AND role_id = ?", "t1", role.ID))
	require.Equal(t, int64(1), countProvisionRows(t, db, &model.UserRoleEntity{}, "tenant_id = ? AND user_id = ? AND role_id = ?", "t1", "u1", role.ID))

	// 租户隔离：不得污染其他租户
	require.Equal(t, int64(0), countProvisionRows(t, db, &model.RoleEntity{}, "tenant_id = ?", "t2"))
}

func TestProvisionTenantAdmin_Idempotent(t *testing.T) {
	db := newProvisionTestDB(t)
	app := seedTenantAdminApp(t, db, ProvisionMenuCodes...)
	ctx := context.Background()
	req := &ProvisionTenantAdminReq{TenantID: "t1", GrantUserID: "u1"}

	first, err := ProvisionTenantAdmin(ctx, db, req)
	require.NoError(t, err)
	second, err := ProvisionTenantAdmin(ctx, db, req)
	require.NoError(t, err)

	// 重复执行必须命中同一角色，且各表行数不变（四张表均无唯一索引，靠应用层查重）
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, int64(1), countProvisionRows(t, db, &model.TenantApplicationEntity{}, "tenant_id = ? AND app_id = ?", "t1", app.ID))
	require.Equal(t, int64(1), countProvisionRows(t, db, &model.RoleEntity{}, "tenant_id = ?", "t1"))
	require.Equal(t, int64(len(ProvisionMenuCodes)), countProvisionRows(t, db, &model.RoleMenuEntity{}, "tenant_id = ?", "t1"))
	require.Equal(t, int64(1), countProvisionRows(t, db, &model.UserRoleEntity{}, "tenant_id = ?", "t1"))
}

// TestProvisionTenantAdmin_BackfillsAdminType 存量库内置角色被误置为普通类型时，
// 权限开通应命中同一行（(tenant_id, app_id, source=builtin) 即内置角色幂等键）并回填 admin_type。
func TestProvisionTenantAdmin_BackfillsAdminType(t *testing.T) {
	db := newProvisionTestDB(t)
	app := seedTenantAdminApp(t, db, ProvisionMenuCodes...)
	ctx := context.Background()

	// 存量脏数据：内置角色被改错系统管理类型
	legacy := &model.RoleEntity{TenantID: "t1", AppID: app.ID, Name: ProvisionRoleName,
		Source: string(model.RoleSourceBuiltin), AdminType: model.SysAdminTypeNormal}
	require.NoError(t, db.WithContext(ctx).Create(legacy).Error)

	role, err := ProvisionTenantAdmin(ctx, db, &ProvisionTenantAdminReq{TenantID: "t1"})
	require.NoError(t, err)
	require.Equal(t, legacy.ID, role.ID)
	require.Equal(t, ProvisionAdminType, role.AdminType)
}

func TestProvisionTenantAdmin_ApplicationMissing(t *testing.T) {
	db := newProvisionTestDB(t)
	_, err := ProvisionTenantAdmin(context.Background(), db, &ProvisionTenantAdminReq{TenantID: "t1"})
	require.Error(t, err, "应用（全局种子数据）缺失必须报错让调用方回滚，不得静默跳过")
}

func TestProvisionTenantAdmin_MissingMenuSkipsGrant(t *testing.T) {
	db := newProvisionTestDB(t)
	// 只种 3 个菜单（少一个 tenant-api-key）
	app := seedTenantAdminApp(t, db, ProvisionMenuCodes[:3]...)

	role, err := ProvisionTenantAdmin(context.Background(), db, &ProvisionTenantAdminReq{TenantID: "t1"})
	require.NoError(t, err, "缺失菜单只应告警跳过，不阻断建租户")
	require.Equal(t, int64(3), countProvisionRows(t, db, &model.RoleMenuEntity{}, "tenant_id = ? AND role_id = ?", "t1", role.ID))
	require.Equal(t, int64(1), countProvisionRows(t, db, &model.TenantApplicationEntity{}, "tenant_id = ? AND app_id = ?", "t1", app.ID))
}

func TestProvisionTenantAdmin_NoGrantUserSkipsUserRole(t *testing.T) {
	db := newProvisionTestDB(t)
	seedTenantAdminApp(t, db, ProvisionMenuCodes...)

	_, err := ProvisionTenantAdmin(context.Background(), db, &ProvisionTenantAdminReq{TenantID: "t1"})
	require.NoError(t, err)
	require.Equal(t, int64(0), countProvisionRows(t, db, &model.UserRoleEntity{}, "tenant_id = ?", "t1"))
}

func TestProvisionTenantAdmin_NilTxOrTenant(t *testing.T) {
	db := newProvisionTestDB(t)
	_, err := ProvisionTenantAdmin(context.Background(), nil, &ProvisionTenantAdminReq{TenantID: "t1"})
	require.Error(t, err)
	_, err = ProvisionTenantAdmin(context.Background(), db, &ProvisionTenantAdminReq{})
	require.Error(t, err)
}
