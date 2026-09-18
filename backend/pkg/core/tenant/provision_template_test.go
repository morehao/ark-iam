package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// templateJSON 构造 application.role_template 列值（应用侧写入的形状）。
func templateJSON(t *testing.T, items ...model.RoleTemplateItem) datatypes.JSON {
	t.Helper()
	raw, err := json.Marshal(model.RoleTemplateItemList(items))
	require.NoError(t, err)
	return datatypes.JSON(raw)
}

// seedSubscribedTenant 播种一条租户订阅（not null JSON 列显式给值，sqlite 不接受 NULL）。
func seedSubscribedTenant(t *testing.T, db *gorm.DB, tenantID, appID string) {
	t.Helper()
	require.NoError(t, db.WithContext(context.Background()).Create(&model.TenantApplicationEntity{
		TenantID:     tenantID,
		AppID:        appID,
		Status:       model.TenantApplicationStatusEnable,
		Config:       datatypes.JSON("{}"),
		GrantedScope: datatypes.JSON("[]"),
	}).Error)
}

// countTemplateRoles 按 (租户, 应用, 来源) 统计角色行数（跨租户读，测试直连库无需租户作用域）。
func countTemplateRoles(t *testing.T, db *gorm.DB, tenantID, appID string, source model.RoleSource) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&model.RoleEntity{}).
		Where("tenant_id = ? AND app_id = ? AND source = ?", tenantID, appID, source).
		Count(&n).Error)
	return n
}

// findTemplateRole 查某租户某应用下的单个角色（按编码），不存在返回 nil。
func findTemplateRole(t *testing.T, db *gorm.DB, tenantID, appID string, code model.RoleCode) *model.RoleEntity {
	t.Helper()
	var role model.RoleEntity
	err := db.Where("tenant_id = ? AND app_id = ? AND code = ?", tenantID, appID, code).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	require.NoError(t, err)
	return &role
}

// TestSyncAppRoleTemplateMaterializeAndWithdraw 模板物化/撤下的完整闭环：
//
//  1. 订阅了应用的所有租户都拿到模板角色（source=builtin），重复同步不产生重复行；
//  2. 模板改名回写到已物化的角色；
//  3. 模板移除的条目在租户侧被**真撤下**（连 user_role / role_menu 关联一起清理）——
//     否则被撤的 code 仍在 ID token 的 groups 里，继续拿到下游策略；
//  4. 产品锚点编码不吃撤下逻辑（tenant_admin 是产品锚点，模板里没有它也必须留存）；
//  5. 已被自建角色占用的编码不被劫持（防御存量数据：新模型下租户已无法写入 code）。
func TestSyncAppRoleTemplateMaterializeAndWithdraw(t *testing.T) {
	db := newProvisionTestDB(t)
	ctx := context.Background()

	app := &model.ApplicationEntity{
		Code:    "rustfs",
		SeedKey: "rustfs",
		Name:    "对象存储",
		Status:  model.AppStatusEnable,
		RoleTemplate: templateJSON(t,
			model.RoleTemplateItem{Code: "storage_admin", Name: "存储管理员"},
			model.RoleTemplateItem{Code: "readonly", Name: "只读用户"},
		),
	}
	require.NoError(t, db.WithContext(ctx).Create(app).Error)
	seedSubscribedTenant(t, db, "t1", app.ID)
	seedSubscribedTenant(t, db, "t2", app.ID)

	// t1 的 readonly 已被自建角色占用：同步不得劫持它（改名/改来源），只跳过该条
	require.NoError(t, db.WithContext(ctx).Create(&model.RoleEntity{
		TenantID: "t1", AppID: app.ID, Code: "readonly", Name: "租户自建只读",
		Source: model.RoleSourceCustom, AdminType: model.SysAdminTypeNormal,
	}).Error)
	// 产品锚点：模板未声明 tenant_admin，撤下逻辑必须跳过它
	require.NoError(t, db.WithContext(ctx).Create(&model.RoleEntity{
		TenantID: "t1", AppID: app.ID, Code: model.RoleCodeTenantAdmin, Name: "租户管理员",
		Source: model.RoleSourceBuiltin, AdminType: model.SysAdminTypeAdmin,
	}).Error)

	require.NoError(t, SyncAppRoleTemplateToTenants(ctx, db, &SyncAppRoleTemplateToTenantsReq{AppID: app.ID, CreatedBy: "op"}))

	// 1. t1：storage_admin + readonly(custom) + tenant_admin 锚点 = 2 builtin；t2：2 builtin
	require.Equal(t, int64(2), countTemplateRoles(t, db, "t1", app.ID, model.RoleSourceBuiltin))
	require.Equal(t, int64(2), countTemplateRoles(t, db, "t2", app.ID, model.RoleSourceBuiltin))
	got := findTemplateRole(t, db, "t1", app.ID, "storage_admin")
	require.NotNil(t, got)
	require.Equal(t, "存储管理员", got.Name)
	occupied := findTemplateRole(t, db, "t1", app.ID, "readonly")
	require.NotNil(t, occupied)
	require.Equal(t, model.RoleSourceCustom, occupied.Source)
	require.Equal(t, "租户自建只读", occupied.Name)

	// 2. 幂等：重复同步不新增行
	require.NoError(t, SyncAppRoleTemplateToTenants(ctx, db, &SyncAppRoleTemplateToTenantsReq{AppID: app.ID, CreatedBy: "op"}))
	require.Equal(t, int64(2), countTemplateRoles(t, db, "t1", app.ID, model.RoleSourceBuiltin))
	require.Equal(t, int64(2), countTemplateRoles(t, db, "t2", app.ID, model.RoleSourceBuiltin))

	// 3. 改名回写
	app.RoleTemplate = templateJSON(t,
		model.RoleTemplateItem{Code: "storage_admin", Name: "存储超级管理员"},
		model.RoleTemplateItem{Code: "readonly", Name: "只读用户"},
	)
	require.NoError(t, db.WithContext(ctx).Model(&model.ApplicationEntity{}).
		Where("id = ?", app.ID).Update("role_template", app.RoleTemplate).Error)
	require.NoError(t, SyncAppRoleTemplateToTenants(ctx, db, &SyncAppRoleTemplateToTenantsReq{AppID: app.ID, CreatedBy: "op"}))
	require.Equal(t, "存储超级管理员", findTemplateRole(t, db, "t2", app.ID, "storage_admin").Name)

	// 4. 撤下：模板只留 storage_admin，readonly 在 t2 被删除（连带关联），t1 的自建只读不受影响
	withdrawn := findTemplateRole(t, db, "t2", app.ID, "readonly")
	require.NotNil(t, withdrawn)
	require.NoError(t, db.WithContext(ctx).Create(&model.MenuEntity{
		AppID: app.ID, Name: "对象列表", Code: "object_list", SeedKey: "object_list",
		Path: "/objects", Type: model.MenuTypeMenu, Visibility: model.MenuVisibilityAdmin, Status: model.MenuStatusEnable,
	}).Error)
	var menu model.MenuEntity
	require.NoError(t, db.WithContext(ctx).Where("code = ?", "object_list").First(&menu).Error)
	require.NoError(t, db.WithContext(ctx).Create(&model.UserRoleEntity{TenantID: "t2", UserID: "u1", RoleID: withdrawn.ID}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&model.RoleMenuEntity{TenantID: "t2", RoleID: withdrawn.ID, MenuID: menu.ID}).Error)

	app.RoleTemplate = templateJSON(t, model.RoleTemplateItem{Code: "storage_admin", Name: "存储超级管理员"})
	require.NoError(t, db.WithContext(ctx).Model(&model.ApplicationEntity{}).
		Where("id = ?", app.ID).Update("role_template", app.RoleTemplate).Error)
	require.NoError(t, SyncAppRoleTemplateToTenants(ctx, db, &SyncAppRoleTemplateToTenantsReq{AppID: app.ID, CreatedBy: "op"}))

	require.Nil(t, findTemplateRole(t, db, "t2", app.ID, "readonly"))
	require.Equal(t, int64(1), countTemplateRoles(t, db, "t2", app.ID, model.RoleSourceBuiltin))
	var userRoleCount, roleMenuCount int64
	require.NoError(t, db.Model(&model.UserRoleEntity{}).Where("role_id = ?", withdrawn.ID).Count(&userRoleCount).Error)
	require.NoError(t, db.Model(&model.RoleMenuEntity{}).Where("role_id = ?", withdrawn.ID).Count(&roleMenuCount).Error)
	require.Zero(t, userRoleCount)
	require.Zero(t, roleMenuCount)
	// 5. 产品锚点留存；t1 的自建只读未被劫持也未被删除
	require.NotNil(t, findTemplateRole(t, db, "t1", app.ID, model.RoleCodeTenantAdmin))
	require.NotNil(t, findTemplateRole(t, db, "t1", app.ID, "readonly"))
	require.Equal(t, model.RoleSourceCustom, findTemplateRole(t, db, "t1", app.ID, "readonly").Source)
	// t1：storage_admin + tenant_admin 锚点 = 2 builtin（自建只读不计入）
	require.Equal(t, int64(2), countTemplateRoles(t, db, "t1", app.ID, model.RoleSourceBuiltin))
}

// TestSyncAppRoleTemplateSkipsProductAnchor 产品锚点编码永不被模板定义/覆盖。
//
// 写入侧（svcapplication）已拒绝把锚点写进模板，但 role_template 是普通 JSON 列——直连改库、
// 历史数据或未来的写入路径都可能绕过该校验，故同步侧再挡一次：模板里出现 platform_admin /
// tenant_admin 时跳过该条，既不新建同码角色、也不改名既有锚点。
func TestSyncAppRoleTemplateSkipsProductAnchor(t *testing.T) {
	db := newProvisionTestDB(t)
	ctx := context.Background()

	app := &model.ApplicationEntity{
		Code:    "rustfs",
		SeedKey: "rustfs",
		Name:    "对象存储",
		Status:  model.AppStatusEnable,
		// 直接构造含锚点的模板（绕过 service 校验）
		RoleTemplate: templateJSON(t,
			model.RoleTemplateItem{Code: model.RoleCodeTenantAdmin, Name: "伪装租户管理员"},
			model.RoleTemplateItem{Code: model.RoleCodePlatformAdmin, Name: "伪装平台管理员"},
			model.RoleTemplateItem{Code: "storage_admin", Name: "存储管理员"},
		),
	}
	require.NoError(t, db.WithContext(ctx).Create(app).Error)
	seedSubscribedTenant(t, db, "t1", app.ID)
	require.NoError(t, db.WithContext(ctx).Create(&model.RoleEntity{
		TenantID: "t1", AppID: app.ID, Code: model.RoleCodeTenantAdmin, Name: "租户管理员",
		Source: model.RoleSourceBuiltin, AdminType: model.SysAdminTypeAdmin,
	}).Error)

	require.NoError(t, SyncAppRoleTemplateToTenants(ctx, db, &SyncAppRoleTemplateToTenantsReq{AppID: app.ID, CreatedBy: "op"}))

	// 锚点角色原名留存、未被模板改名，且没有为锚点编码新建第二行
	anchor := findTemplateRole(t, db, "t1", app.ID, model.RoleCodeTenantAdmin)
	require.NotNil(t, anchor)
	require.Equal(t, "租户管理员", anchor.Name)
	require.Nil(t, findTemplateRole(t, db, "t1", app.ID, model.RoleCodePlatformAdmin))
	// 非锚点条目照常物化
	require.NotNil(t, findTemplateRole(t, db, "t1", app.ID, "storage_admin"))
	// t1：storage_admin + tenant_admin 锚点 = 2
	require.Equal(t, int64(2), countTemplateRoles(t, db, "t1", app.ID, model.RoleSourceBuiltin))
}
