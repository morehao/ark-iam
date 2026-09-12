// Package seed 提供 IAM 基础种子数据的幂等写入能力。
//
// 替代历史 MySQL 方言建表/种子脚本（scripts/sql/*.sql 已废弃删除）：服务启动时
// 基于唯一键（code / client_id / username 等）查重，已存在则跳过、不存在则创建，
// 因此可安全重复执行，兼容全新数据库与已有数据的升级场景。
// 自 string-id 改造起所有主键为字符串（UUID v7），实体间关联在写入时动态接线，
// 不再依赖固定的数字主键。
//
// 业务约束：用户必须从属于某个部门（部门节点），种子管理员同样从属于
// 租户的顶级部门（根部门节点，seedRootDepartment 创建），归属关系为
// member 行政主部门。
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/password"
	// 别名：SeedIam 内以 tenant 命名的局部变量会遮蔽同名包
	iamtenant "github.com/morehao/ark-iam/pkg/iam/tenant"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

const (
	// tenantCodePlatform 平台租户（Default Tenant）编码：固定值，形态与自动生成规则
	// （pkg/iam/tenant.GenerateCode：t_<12 位随机 hex>）一致——同前缀、后缀可读且固定。
	// 自动生成编码的随机段只用小写 hex，"platform" 含非 hex 字符，故两者永不冲突。
	tenantCodePlatform = "t_platform"
	// tenantCodePlatformLegacy 历史种子编码（旧版本为 "platform"）。
	// 启动时若命中该编码的平台租户，原地改名（保留主键，避免租户重建导致引用失联）。
	tenantCodePlatformLegacy = "platform"

	// 应用编码规则：小写字母开头，仅含小写字母/数字/下划线（model.AppCodePattern），
	// 与自动生成的租户编码（t_<hex>）同为下划线连接。
	appCodeAdmin       = "platform_admin"
	appCodeTenantAdmin = "tenant_admin"

	// appCodeAdminLegacy / appCodeTenantAdminLegacy 历史种子编码（旧版本为连字符形态）。
	// 启动时若命中旧编码，原地改名（保留主键，避免改编码规则后重复建出第二个内置应用——
	// 菜单、租户订阅、角色都挂在 app_id 上）。
	appCodeAdminLegacy       = "platform-admin"
	appCodeTenantAdminLegacy = "tenant-admin"

	oauthClientPlatformAdminWeb = "platform-admin-web"
	oauthClientTenantAdminWeb   = "tenant-admin-web"
)

// seedMenu 菜单种子定义；parentCode 为空表示顶级菜单。visibility 缺省为 public。
type seedMenu struct {
	appCode    string
	parentCode string
	name       string
	code       string
	path       string
	icon       string
	sort       int
	component  string
	menuType   model.MenuType
	visibility model.MenuVisibility
}

// SeedIam 幂等写入 IAM 基础种子数据。任一环节失败即返回错误，由调用方决定是否阻断启动。
func SeedIam(ctx context.Context, db *gorm.DB) error {
	// 1. 平台租户
	tenant, err := getOrCreateTenant(ctx, db)
	if err != nil {
		return err
	}

	// 2. 租户同名顶级部门（用户归属的根部门，管理员也归属于此）
	rootDept, err := seedRootDepartment(ctx, db, tenant)
	if err != nil {
		return err
	}

	// 3. 应用（历史库的连字符编码由 getOrCreateApplication 原地改名）
	adminApp, err := getOrCreateApplication(ctx, db, appCodeAdmin, appCodeAdminLegacy, "管理后台", "平台管理后台应用", 0, model.AppSourceBuiltin)
	if err != nil {
		return err
	}
	tenantAdminApp, err := getOrCreateApplication(ctx, db, appCodeTenantAdmin, appCodeTenantAdminLegacy, "租户自服务", "租户自服务控制台应用", 1, model.AppSourceBuiltin)
	if err != nil {
		return err
	}

	// 4. 角色（admin 归属管理后台；tenant_admin 由第 10 步的权限开通统一创建）
	adminRole, err := seedRoles(ctx, db, tenant, adminApp)
	if err != nil {
		return err
	}

	// 5. 菜单
	menus, err := seedMenus(ctx, db, adminApp, tenantAdminApp)
	if err != nil {
		return err
	}

	// 6. 已下线菜单清理（菜单行 + role_menu 授权绑定）
	if err := pruneRetiredMenus(ctx, db, map[string]*model.ApplicationEntity{
		appCodeAdmin:       adminApp,
		appCodeTenantAdmin: tenantAdminApp,
	}); err != nil {
		return err
	}

	// 7. 角色-菜单关联（仅管理后台 admin 角色；tenant_admin 由第 10 步开通时授权）
	if err := seedRoleMenus(ctx, db, tenant, adminRole, menus); err != nil {
		return err
	}

	// 8. 租户应用订阅（管理后台 platform_admin；租户自服务 tenant_admin 由第 10 步开通时订阅）
	if err := seedTenantApplications(ctx, db, tenant, adminApp); err != nil {
		return err
	}

	// 9. 默认管理员（person + user + 顶级部门归属）
	adminUser, err := seedAdminUser(ctx, db, tenant, rootDept)
	if err != nil {
		return err
	}
	if err := seedAdminUserRole(ctx, db, tenant, adminUser, adminRole); err != nil {
		return err
	}

	// 10. 平台租户的租户自服务权限开通：与"新建租户"共用同一实现
	// （pkg/iam/tenant.ProvisionTenantAdmin），保证内置角色/菜单授权/订阅只有一份定义。
	if _, err := iamtenant.ProvisionTenantAdmin(ctx, db, &iamtenant.ProvisionTenantAdminReq{
		TenantID:    tenant.ID,
		GrantUserID: adminUser.ID,
	}); err != nil {
		return fmt.Errorf("seed provision tenant admin fail: %w", err)
	}

	// 11. OIDC 测试客户端
	if err := seedOIDCClients(ctx, db, tenant, adminApp); err != nil {
		return err
	}

	return nil
}

// ---------- 各实体种子实现 ----------

// findTenantByCode 按编码查平台租户；不存在返回 (nil, nil)，系统错误返回 (nil, err)。
func findTenantByCode(db *gorm.DB, code string) (*model.TenantEntity, error) {
	entity := &model.TenantEntity{}
	err := db.Where("code = ?", code).First(entity).Error
	if err == nil {
		return entity, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("seed tenant query fail (code=%s): %w", code, err)
}

func getOrCreateTenant(ctx context.Context, db *gorm.DB) (*model.TenantEntity, error) {
	entity, err := findTenantByCode(db, tenantCodePlatform)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		// 历史库平台租户编码为 tenantCodePlatformLegacy（"platform"）：原地改名，
		// 保留主键，避免改编码规则后重复建出第二个平台租户。仅迁移平台类型租户，
		// 防止误改恰好同名的客户租户。
		legacy, lErr := findTenantByCode(db, tenantCodePlatformLegacy)
		if lErr != nil {
			return nil, lErr
		}
		if legacy != nil && legacy.Type == model.TenantTypePlatform {
			if uErr := db.Model(&model.TenantEntity{}).Where("id = ?", legacy.ID).
				Update("code", tenantCodePlatform).Error; uErr != nil {
				return nil, fmt.Errorf("seed tenant code migrate fail: %w", uErr)
			}
			legacy.Code = tenantCodePlatform
			glog.Infof(ctx, "[seed] tenant code migrated (%s -> %s), id:%s",
				tenantCodePlatformLegacy, tenantCodePlatform, legacy.ID)
			entity = legacy
		}
	}
	if entity != nil {
		// 幂等回填：平台租户是平台控制台自身的租户，被挂起会导致整个控制台失联，
		// 存量库若状态异常，种子启动时纠正为 active。
		if entity.Status != model.TenantStatusActive {
			if uErr := db.Model(&model.TenantEntity{}).Where("id = ?", entity.ID).
				Update("status", model.TenantStatusActive).Error; uErr != nil {
				return nil, fmt.Errorf("seed tenant status backfill fail: %w", uErr)
			}
			entity.Status = model.TenantStatusActive
			glog.Infof(ctx, "[seed] tenant status backfilled to active, id:%s", entity.ID)
		}
		return entity, nil
	}
	entity = &model.TenantEntity{
		Code:   tenantCodePlatform,
		Name:   "Default Tenant",
		Type:   model.TenantTypePlatform,
		DbUser: "default_user",
		Status: model.TenantStatusActive,
		Tag:    "default",
	}
	if err := db.WithContext(ctx).Create(entity).Error; err != nil {
		return nil, fmt.Errorf("seed tenant create fail: %w", err)
	}
	glog.Infof(ctx, "[seed] tenant created, id:%s code:%s", entity.ID, entity.Code)
	return entity, nil
}

// seedRootDepartment 确保租户存在唯一顶级部门（根部门节点），并返回该节点。
// 所有种子用户（含管理员）均从属于此顶级部门。
func seedRootDepartment(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity) (*model.DepartmentEntity, error) {
	dept := &model.DepartmentEntity{}
	err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(dept).Error
	if err == nil {
		return dept, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed root department query fail: %w", err)
	}
	dept = &model.DepartmentEntity{
		TenantID: tenant.ID,
		Name:     tenant.Name,
		Status:   model.DeptNodeStatusEnable,
	}
	if err := db.WithContext(ctx).Create(dept).Error; err != nil {
		return nil, fmt.Errorf("seed root department create fail: %w", err)
	}
	// 根节点路径："/"+id，深度 1（ID 由 BeforeCreate 生成，需创建后补写）
	if err := db.WithContext(ctx).Model(dept).Updates(map[string]any{
		"dept_path":  "/" + dept.ID,
		"dept_depth": 1,
	}).Error; err != nil {
		return nil, fmt.Errorf("seed root department path fail: %w", err)
	}
	return dept, nil
}

// findApplicationByCode 按编码查应用；不存在返回 (nil, nil)，系统错误返回 (nil, err)。
func findApplicationByCode(db *gorm.DB, code string) (*model.ApplicationEntity, error) {
	entity := &model.ApplicationEntity{}
	err := db.Where("code = ?", code).First(entity).Error
	if err == nil {
		return entity, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("seed application query fail (code=%s): %w", code, err)
}

// getOrCreateApplication 幂等获取内置应用。
// legacyCode 非空时，把历史库的旧编码（连字符形态）原地改名为新编码：
// 编码是应用的业务唯一键，改名不影响任何以 app_id 关联的菜单/订阅/角色。
func getOrCreateApplication(ctx context.Context, db *gorm.DB, code, legacyCode, name, desc string, sort int, source model.AppSource) (*model.ApplicationEntity, error) {
	entity, err := findApplicationByCode(db, code)
	if err != nil {
		return nil, err
	}
	if legacyCode != "" {
		legacy, lErr := findApplicationByCode(db, legacyCode)
		if lErr != nil {
			return nil, lErr
		}
		switch {
		case legacy == nil:
			// 正常路径：旧编码不存在（全新库或已迁移过）
		case entity != nil:
			// 新旧编码并存：无法判断哪一行才是内置应用。此时回填 source 会把用户自建应用
			// 改写成内置（获得删除保护并接管菜单范围），故宁可中断启动，交人工确认后删除其一。
			return nil, fmt.Errorf("seed application code conflict: %q 与 %q 同时存在，请人工确认哪一行是内置应用并删除另一行", code, legacyCode)
		default:
			// 历史库编码迁移（platform-admin -> platform_admin）
			if uErr := db.WithContext(ctx).Model(&model.ApplicationEntity{}).Where("id = ?", legacy.ID).
				Update("code", code).Error; uErr != nil {
				return nil, fmt.Errorf("seed application code migrate fail (%s -> %s): %w", legacyCode, code, uErr)
			}
			legacy.Code = code
			glog.Infof(ctx, "[seed] application code migrated (%s -> %s), id:%s", legacyCode, code, legacy.ID)
			entity = legacy
		}
	}
	if entity != nil {
		// 幂等回填 source：存量库的 source 是 AutoMigrate 补列时按列默认值 third_party 落下的，
		// 会把管理后台/租户自服务误判为第三方接入（前者丢删除保护、后者菜单还会串进租户控制台），
		// 故种子启动时按定义原地纠正。只回填 source —— name/description/sort/status 在控制台可改，
		// 种子不得覆盖运维改动。
		if entity.Source != source {
			if uErr := db.WithContext(ctx).Model(&model.ApplicationEntity{}).Where("id = ?", entity.ID).
				Update("source", source).Error; uErr != nil {
				return nil, fmt.Errorf("seed application %s source backfill fail: %w", code, uErr)
			}
			glog.Infof(ctx, "[seed] application source backfilled (%s -> %s), code:%s", entity.Source, source, code)
			entity.Source = source
		}
		return entity, nil
	}
	entity = &model.ApplicationEntity{
		Code:        code,
		Name:        name,
		Description: desc,
		Source:      source,
		Status:      model.AppStatusEnable,
		Sort:        sort,
	}
	if err := db.WithContext(ctx).Create(entity).Error; err != nil {
		return nil, fmt.Errorf("seed application %s create fail: %w", code, err)
	}
	glog.Infof(ctx, "[seed] application created, id:%s code:%s", entity.ID, code)
	return entity, nil
}

// seedRoles 只种管理后台 admin 角色：租户自服务的 tenant_admin 由权限开通统一创建
// （pkg/iam/tenant.ProvisionTenantAdmin），建租户与种子共用同一份角色/授权定义。
// 角色无业务编码，内置角色以 (tenant_id, app_id, source=builtin) 为幂等定位键。
func seedRoles(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, adminApp *model.ApplicationEntity) (*model.RoleEntity, error) {
	// 系统管理类型是内置角色的显式声明，不设隐式缺省：未声明/非法值直接失败，
	// 避免内置管理员角色被静默播种成普通类型（IsBuiltinAdmin 依赖 admin_type=admin）。
	adminType := iamtenant.ProvisionAdminType
	switch adminType {
	case model.SysAdminTypeAdmin, model.SysAdminTypeNormal:
	default:
		return nil, fmt.Errorf("seed admin role: 非法系统管理类型 %q(必须显式声明 admin/normal)", adminType)
	}

	entity := &model.RoleEntity{}
	err := db.Where("tenant_id = ? AND app_id = ? AND source = ?", tenant.ID, adminApp.ID, model.RoleSourceBuiltin).First(entity).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed admin role query fail: %w", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		entity = &model.RoleEntity{
			TenantID:    tenant.ID,
			AppID:       adminApp.ID,
			Name:        "管理员",
			Description: "系统管理员，拥有所有权限",
			Source:      model.RoleSourceBuiltin,
			AdminType:   adminType,
		}
		if err := db.WithContext(ctx).Create(entity).Error; err != nil {
			return nil, fmt.Errorf("seed admin role create fail: %w", err)
		}
		return entity, nil
	}
	// 幂等回填：存量内置角色的 admin_type 随种子定义更新
	if entity.AdminType != adminType {
		if uerr := db.Model(&model.RoleEntity{}).Where("id = ?", entity.ID).
			Update("admin_type", adminType).Error; uerr != nil {
			return nil, fmt.Errorf("seed admin role update fail: %w", uerr)
		}
		entity.AdminType = adminType
	}
	return entity, nil
}

// menuTypeOf 返回菜单种子定义的 type；未显式指定时缺省为 menu。
func menuTypeOf(def seedMenu) model.MenuType {
	if def.menuType == "" {
		return model.MenuTypeMenu
	}
	return def.menuType
}

func seedMenus(ctx context.Context, db *gorm.DB, adminApp, tenantAdminApp *model.ApplicationEntity) (map[string]*model.MenuEntity, error) {
	defs := []seedMenu{
		// 平台管理控制台：目录分组（type=directory，无页面）+ 页面叶子（type=menu，指向真实前端页面）。
		// 一级菜单按「对象域」划分（对象名词 + 中心/叶子），不使用「X 与 Y」并列命名：
		// 租户中心（租户及其资源）/ 应用中心（应用及其接入凭证）/
		// 平台管理（平台自身治理：菜单字典与审计日志）。
		// 用户与角色不再有平台端入口：两者按租户归属，读写与成员管理全部收敛到租户自服务控制台。
		{appCode: appCodeAdmin, name: "工作台", code: "dashboard", path: "/dashboard", icon: "dashboard", sort: 1, component: "/dashboard/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityMember},
		{appCode: appCodeAdmin, name: "租户中心", code: "grp-tenant", icon: "apartment", sort: 2, menuType: model.MenuTypeDirectory, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "grp-tenant", name: "租户管理", code: "tenant", path: "/tenant", icon: "global", sort: 1, component: "/tenant/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "grp-tenant", name: "租户应用", code: "tenant-application", path: "/tenant-application", icon: "shopping", sort: 2, component: "/tenantApplication/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "grp-tenant", name: "自定义域名", code: "domain", path: "/domain", icon: "global", sort: 3, component: "/domain/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, name: "应用中心", code: "grp-app", icon: "app", sort: 3, menuType: model.MenuTypeDirectory, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "grp-app", name: "应用管理", code: "application", path: "/application", icon: "app", sort: 1, component: "/application/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "grp-app", name: "OAuth客户端", code: "oauth-client", path: "/oauth-client", icon: "key", sort: 2, component: "/oauthClient/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, name: "平台管理", code: "grp-platform", icon: "setting", sort: 4, menuType: model.MenuTypeDirectory, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "grp-platform", name: "菜单管理", code: "menu", path: "/menu", icon: "menu", sort: 1, component: "/menu/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "grp-platform", name: "审计日志", code: "log", path: "/log", icon: "file", sort: 2, component: "/log/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		// 租户自服务一级菜单：控制台定位为「租户管理层专用」（部门/用户/角色/密钥均属管理操作，
		// 全部 visibility=admin 硬隔离；普通成员不面向该控制台，仅内置管理员角色可见与授权）。
		// 用户/角色/密钥编码加 tenant- 前缀，避免与平台菜单 code 撞名。
		{appCode: appCodeTenantAdmin, name: "部门管理", code: "department", path: "/department", icon: "apartment", sort: 1, component: "pages/department", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeTenantAdmin, name: "用户管理", code: "tenant-user", path: "/user", icon: "user", sort: 2, component: "pages/user", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeTenantAdmin, name: "角色管理", code: "tenant-role", path: "/role", icon: "role", sort: 3, component: "pages/role", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeTenantAdmin, name: "API密钥", code: "tenant-api-key", path: "/api-key", icon: "key", sort: 4, component: "pages/apiKey", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	}

	appByCode := map[string]*model.ApplicationEntity{appCodeAdmin: adminApp, appCodeTenantAdmin: tenantAdminApp}
	out := make(map[string]*model.MenuEntity, len(defs))
	for _, def := range defs {
		app := appByCode[def.appCode]
		entity := &model.MenuEntity{}
		err := db.Where("app_id = ? AND code = ?", app.ID, def.code).First(entity).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("seed menu %s query fail: %w", def.code, err)
		}
		parentID := ""
		if def.parentCode != "" {
			parent, ok := out[def.parentCode]
			if !ok {
				return nil, fmt.Errorf("seed menu %s parent %s not found", def.code, def.parentCode)
			}
			parentID = parent.ID
		}
		visibility := def.visibility
		if visibility == "" {
			visibility = model.MenuVisibilityPublic
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entity = &model.MenuEntity{
				AppID:      app.ID,
				ParentID:   parentID,
				Name:       def.name,
				Code:       def.code,
				Path:       def.path,
				Icon:       def.icon,
				Sort:       def.sort,
				Type:       menuTypeOf(def),
				Visibility: visibility,
				Component:  def.component,
				Status:     model.MenuStatusEnable,
			}
			if err := db.WithContext(ctx).Create(entity).Error; err != nil {
				return nil, fmt.Errorf("seed menu %s create fail: %w", def.code, err)
			}
		} else {
			// 幂等回填：存量菜单的关键字段随种子定义更新，保证跨版本升级后与 seed 定义一致。
			updateMap := map[string]any{}
			if entity.Name != def.name {
				updateMap["name"] = def.name
			}
			if entity.Path != def.path {
				updateMap["path"] = def.path
			}
			if entity.Icon != def.icon {
				updateMap["icon"] = def.icon
			}
			if entity.Sort != def.sort {
				updateMap["sort"] = def.sort
			}
			if entity.Component != def.component {
				updateMap["component"] = def.component
			}
			if entity.Visibility != visibility {
				updateMap["visibility"] = visibility
			}
			if entity.Type != menuTypeOf(def) {
				updateMap["type"] = menuTypeOf(def)
			}
			if entity.ParentID != parentID {
				updateMap["parent_id"] = parentID
			}
			if len(updateMap) > 0 {
				if uerr := db.Model(&model.MenuEntity{}).Where("id = ?", entity.ID).
					Updates(updateMap).Error; uerr != nil {
					return nil, fmt.Errorf("seed menu %s update fail: %w", def.code, uerr)
				}
			}
		}
		out[def.code] = entity
	}
	return out, nil
}

// seedRoleMenus 只处理管理后台 admin 角色的菜单授权；
// tenant_admin 的菜单授权由 pkg/iam/tenant.ProvisionTenantAdmin 统一写入。
func seedRoleMenus(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, adminRole *model.RoleEntity, menus map[string]*model.MenuEntity) error {
	menuCodes := []string{
		"dashboard", "menu", "tenant", "application",
		"tenant-application", "oauth-client", "domain", "log",
		"department", "tenant-user", "tenant-role",
	}
	for _, menuCode := range menuCodes {
		menu := menus[menuCode]
		var count int64
		if err := db.Model(&model.RoleMenuEntity{}).
			Where("tenant_id = ? AND role_id = ? AND menu_id = ?", tenant.ID, adminRole.ID, menu.ID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("seed role_menu count fail: %w", err)
		}
		if count > 0 {
			continue
		}
		rm := &model.RoleMenuEntity{TenantID: tenant.ID, RoleID: adminRole.ID, MenuID: menu.ID}
		if err := db.WithContext(ctx).Create(rm).Error; err != nil {
			return fmt.Errorf("seed role_menu create fail: %w", err)
		}
	}
	return nil
}

// seedTenantApplications 写入管理后台的应用订阅；租户自服务（tenant_admin）的订阅
// 由 pkg/iam/tenant.ProvisionTenantAdmin 统一写入。
func seedTenantApplications(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, apps ...*model.ApplicationEntity) error {
	for _, app := range apps {
		var count int64
		if err := db.Model(&model.TenantApplicationEntity{}).
			Where("tenant_id = ? AND app_id = ?", tenant.ID, app.ID).Count(&count).Error; err != nil {
			return fmt.Errorf("seed tenant_application count fail: %w", err)
		}
		if count > 0 {
			continue
		}
		ta := &model.TenantApplicationEntity{TenantID: tenant.ID, AppID: app.ID, Status: model.TenantApplicationStatusEnable, Config: []byte(`{}`), GrantedScope: []byte(`[]`)}
		if err := db.WithContext(ctx).Create(ta).Error; err != nil {
			return fmt.Errorf("seed tenant_application create fail: %w", err)
		}
	}
	return nil
}

// seedAdminUser 幂等写入默认管理员（person + user），并确保其从属于顶级部门 rootDept
// （primary 行政主部门），满足"用户必须从属于某个部门"的业务约束。
// rootDept 缺失时视为种子数据不完整，直接报错，避免产出无归属用户。
func seedAdminUser(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, rootDept *model.DepartmentEntity) (*model.UserEntity, error) {
	if rootDept == nil || rootDept.ID == "" {
		return nil, fmt.Errorf("seed admin user fail: root department not found")
	}
	passwordHash, err := gcrypto.GeneratePasswordHash(password.BootstrapAdminPassword)
	if err != nil {
		return nil, fmt.Errorf("seed admin password hash fail: %w", err)
	}

	// person：以 username 为唯一键
	person := &model.PersonEntity{}
	pErr := db.Where("username = ?", model.StrPtr("admin")).First(person).Error
	if pErr != nil && !errors.Is(pErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed admin person query fail: %w", pErr)
	}
	if errors.Is(pErr, gorm.ErrRecordNotFound) {
		person = &model.PersonEntity{
			Username:          model.StrPtr("admin"),
			PrimaryEmail:      model.StrPtr("admin@example.com"),
			PrimaryPhone:      model.StrPtr("13800000000"),
			PasswordEncrypted: passwordHash,
			PasswordMethod:    model.PasswordMethodBcrypt,
			Name:              "系统管理员",
			Profile:           []byte(`{}`),
			CustomData:        []byte(`{}`),
		}
		if err := db.WithContext(ctx).Create(person).Error; err != nil {
			return nil, fmt.Errorf("seed admin person create fail: %w", err)
		}
	}

	// user：以 (tenant_id, person_id) 为唯一键
	user := &model.UserEntity{}
	uErr := db.Where("tenant_id = ? AND person_id = ?", tenant.ID, person.ID).First(user).Error
	if uErr != nil && !errors.Is(uErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed admin user query fail: %w", uErr)
	}
	if errors.Is(uErr, gorm.ErrRecordNotFound) {
		now := time.Now()
		user = &model.UserEntity{
			TenantID:    tenant.ID,
			PersonID:    person.ID,
			Name:        "系统管理员",
			Profile:     []byte(`{}`),
			CustomData:  []byte(`{}`),
			Source:      model.UserSourceBuiltin,
			IsOwner:     true,
			IsSuspended: false,
			JoinedAt:    &now,
		}
		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			return nil, fmt.Errorf("seed admin user create fail: %w", err)
		}
		glog.Infof(ctx, "[seed] admin user created, id:%s (default password: %s, must_change_password: false)", user.ID, password.BootstrapAdminPassword)
	}

	// 来源回填（幂等，兼容存量库）：种子管理员是内置管理员，source 必须为 builtin，
	// 平台侧"重置内置管理员密码"依赖该标记定位目标用户。
	if user.Source != model.UserSourceBuiltin {
		if uerr := db.WithContext(ctx).Model(&model.UserEntity{}).
			Where("id = ?", user.ID).Update("source", model.UserSourceBuiltin).Error; uerr != nil {
			return nil, fmt.Errorf("seed admin user source backfill fail: %w", uerr)
		}
		user.Source = model.UserSourceBuiltin
	}

	// 顶级部门归属（幂等，兼容已有库升级：admin 用户已存在但尚无部门归属的场景）
	if err := seedAdminUserDepartment(ctx, db, tenant, user, rootDept); err != nil {
		return nil, err
	}
	return user, nil
}

// seedAdminUserDepartment 幂等建立管理员与顶级部门的行政主部门关系（primary）。
// 该函数独立于用户创建之外执行，保证升级场景（用户已存在、归属缺失）也能补齐。
func seedAdminUserDepartment(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, user *model.UserEntity, rootDept *model.DepartmentEntity) error {
	var count int64
	if err := db.Model(&model.DepartmentUserEntity{}).
		Where("tenant_id = ? AND user_id = ? AND department_id = ? AND relation_type = ?",
			tenant.ID, user.ID, rootDept.ID, model.DeptUserRelationPrimary).
		Count(&count).Error; err != nil {
		return fmt.Errorf("seed admin department count fail: %w", err)
	}
	if count > 0 {
		return nil
	}
	relation := &model.DepartmentUserEntity{
		TenantID:     tenant.ID,
		DepartmentID: rootDept.ID,
		UserID:       user.ID,
		RelationType: model.DeptUserRelationPrimary,
	}
	if err := db.WithContext(ctx).Create(relation).Error; err != nil {
		return fmt.Errorf("seed admin department create fail: %w", err)
	}
	glog.Infof(ctx, "[seed] admin department relation created, user_id:%s dept_id:%s", user.ID, rootDept.ID)
	return nil
}

// seedAdminUserRole 绑定管理后台 admin 角色；默认管理员的 tenant_admin 角色绑定
// 由后续的权限开通（pkg/iam/tenant.ProvisionTenantAdmin）统一写入。
func seedAdminUserRole(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, adminUser *model.UserEntity, adminRole *model.RoleEntity) error {
	// 默认管理员持有管理后台 admin 角色
	var count int64
	if err := db.Model(&model.UserRoleEntity{}).
		Where("tenant_id = ? AND user_id = ? AND role_id = ?", tenant.ID, adminUser.ID, adminRole.ID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("seed user_role count fail: %w", err)
	}
	if count > 0 {
		return nil
	}
	ur := &model.UserRoleEntity{TenantID: tenant.ID, UserID: adminUser.ID, RoleID: adminRole.ID}
	if err := db.WithContext(ctx).Create(ur).Error; err != nil {
		return fmt.Errorf("seed user_role create fail: %w", err)
	}
	return nil
}

// seedOIDCClientGrantTypes 种子 OAuth 客户端授权类型：由 model.GrantType 常量序列化，
// 避免在种子数据里裸写 JSON 字面量导致取值漂移。
var seedOIDCClientGrantTypes = func() []byte {
	b, _ := json.Marshal([]model.GrantType{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken})
	return b
}()

func seedOIDCClients(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, app *model.ApplicationEntity) error {
	type clientDef struct {
		code                 string
		name                 string
		redirectURIs         string
		postLogoutRedirect   string
		backChannelLogoutURI string
	}
	defs := []clientDef{
		{
			code:                 oauthClientPlatformAdminWeb,
			name:                 "IAM管理平台",
			redirectURIs:         `["http://localhost:4001/auth/callback"]`,
			postLogoutRedirect:   `["http://localhost:4001/login"]`,
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/platform",
		},
		{
			code:                 oauthClientTenantAdminWeb,
			name:                 "租户管理平台",
			redirectURIs:         `["http://localhost:4002/auth/callback"]`,
			postLogoutRedirect:   `["http://localhost:4002/login"]`,
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/tenant",
		},
	}
	for _, def := range defs {
		entity := &model.ApplicationClientEntity{}
		err := db.Where("code = ?", def.code).First(entity).Error
		if err == nil {
			// 幂等回填 source：同 getOrCreateApplication，存量库补列默认 third_party，
			// 而种子客户端是平台内置客户端，必须纠正为 builtin，否则失去删除保护。
			// 其余字段（回调地址/名称等）在控制台可改，种子不覆盖。
			if entity.Source != model.ApplicationClientSourceBuiltin {
				if uErr := db.WithContext(ctx).Model(&model.ApplicationClientEntity{}).Where("id = ?", entity.ID).
					Update("source", model.ApplicationClientSourceBuiltin).Error; uErr != nil {
					return fmt.Errorf("seed oauth client %s source backfill fail: %w", def.code, uErr)
				}
				glog.Infof(ctx, "[seed] oauth client source backfilled (%s -> %s), code:%s",
					entity.Source, model.ApplicationClientSourceBuiltin, def.code)
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("seed oauth client %s query fail: %w", def.code, err)
		}
		entity = &model.ApplicationClientEntity{
			TenantID:                tenant.ID,
			AppID:                   app.ID,
			Code:                    def.code,
			Name:                    def.name,
			RedirectURIs:            []byte(def.redirectURIs),
			PostLogoutRedirectURIs:  []byte(def.postLogoutRedirect),
			BackChannelLogoutURI:    def.backChannelLogoutURI,
			GrantTypes:              seedOIDCClientGrantTypes,
			ResponseTypes:           []byte(`["code"]`),
			TokenEndpointAuthMethod: model.TokenEndpointAuthMethodNone,
			RequirePKCE:             true,
			DefaultScopes:           []byte(`["openid","profile","email"]`),
			Source:                  model.ApplicationClientSourceBuiltin,
			Status:                  model.ApplicationClientStatusEnable,
		}
		if err := db.WithContext(ctx).Create(entity).Error; err != nil {
			return fmt.Errorf("seed oauth client %s create fail: %w", def.code, err)
		}
		glog.Infof(ctx, "[seed] oauth client created, code:%s", def.code)
	}
	return nil
}
