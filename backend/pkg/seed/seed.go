// Package seed 提供 IAM 基础种子数据的幂等写入能力。
//
// 替代历史 scripts/sql/iam_seed_data.sql（MySQL 方言）：服务启动时基于
// 唯一键（code / client_id / username 等）查重，已存在则跳过、不存在则创建，
// 因此可安全重复执行，兼容全新数据库与已有数据的升级场景。
// 自 string-id 改造起所有主键为字符串（UUID v7），实体间关联在写入时动态接线，
// 不再依赖固定的数字主键。
package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

const (
	tenantCodePlatform = "platform"

	adminPassword = "admin123"

	appCodeAdmin       = "admin"
	appCodeTenantAdmin = "tenant-admin"

	resourceIndicatorAdmin = "urn:ark:iam:admin"
	resourceIndicatorMe    = "urn:ark:iam:me"

	menuStatusEnable = "enable"
	menuTypeMenu     = "menu"

	oauthClientPlatformAdminWeb = "platform-admin-web"
	oauthClientTenantAdminWeb   = "tenant-admin-web"
)

// seedRole 角色种子定义。
type seedRole struct {
	code      string
	name      string
	desc      string
	isDefault int8
}

// seedMenu 菜单种子定义；parentCode 为空表示顶级菜单。
type seedMenu struct {
	appCode    string
	parentCode string
	name       string
	code       string
	path       string
	icon       string
	sort       int
	component  string
	permission string
}

// seedScope 权限（scope）种子定义；resourceIndicator 指明归属资源。
type seedScope struct {
	name        string
	description string
	resource    string
}

// SeedIam 幂等写入 IAM 基础种子数据。任一环节失败即返回错误，由调用方决定是否阻断启动。
func SeedIam(ctx context.Context, db *gorm.DB) error {
	// 1. 平台租户
	tenant, err := getOrCreateTenant(ctx, db)
	if err != nil {
		return err
	}

	// 2. 租户同名顶级部门
	if err := seedRootDepartment(ctx, db, tenant); err != nil {
		return err
	}

	// 3. 应用
	adminApp, err := getOrCreateApplication(ctx, db, appCodeAdmin, "管理后台", "平台管理后台应用", 0, 1)
	if err != nil {
		return err
	}
	tenantAdminApp, err := getOrCreateApplication(ctx, db, appCodeTenantAdmin, "租户自服务", "租户自服务控制台应用", 1, 0)
	if err != nil {
		return err
	}

	// 4. 角色
	roles, err := seedRoles(ctx, db, tenant, adminApp)
	if err != nil {
		return err
	}

	// 5. 资源 + 权限
	resources, err := seedResources(ctx, db, tenant)
	if err != nil {
		return err
	}
	scopes, err := seedScopes(ctx, db, tenant, resources)
	if err != nil {
		return err
	}

	// 6. 角色-权限关联
	if err := seedRoleScopes(ctx, db, tenant, roles, scopes); err != nil {
		return err
	}

	// 7. 菜单
	menus, err := seedMenus(ctx, db, adminApp, tenantAdminApp)
	if err != nil {
		return err
	}

	// 8. 角色-菜单关联
	if err := seedRoleMenus(ctx, db, tenant, roles, menus); err != nil {
		return err
	}

	// 9. 租户应用订阅
	if err := seedTenantApplications(ctx, db, tenant, adminApp, tenantAdminApp); err != nil {
		return err
	}

	// 10. 默认管理员（person + user + user_role）
	adminUser, err := seedAdminUser(ctx, db, tenant)
	if err != nil {
		return err
	}
	if err := seedAdminUserRole(ctx, db, tenant, adminUser, roles); err != nil {
		return err
	}

	// 11. OIDC 测试客户端
	if err := seedOIDCClients(ctx, db, tenant, adminApp); err != nil {
		return err
	}

	return nil
}

// ---------- 各实体种子实现 ----------

func getOrCreateTenant(ctx context.Context, db *gorm.DB) (*model.TenantEntity, error) {
	entity := &model.TenantEntity{}
	err := db.Where("code = ?", tenantCodePlatform).First(entity).Error
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed tenant query fail: %w", err)
	}
	entity = &model.TenantEntity{
		Code:        tenantCodePlatform,
		Name:        "Default Tenant",
		Type:        model.TenantTypePlatform,
		DbUser:      "default_user",
		IsSuspended: 0,
		Tag:         "default",
	}
	if err := db.WithContext(ctx).Create(entity).Error; err != nil {
		return nil, fmt.Errorf("seed tenant create fail: %w", err)
	}
	glog.Infof(ctx, "[seed] tenant created, id:%s code:%s", entity.ID, entity.Code)
	return entity, nil
}

func seedRootDepartment(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity) error {
	var count int64
	if err := db.Model(&model.DepartmentEntity{}).
		Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").Count(&count).Error; err != nil {
		return fmt.Errorf("seed root department count fail: %w", err)
	}
	if count > 0 {
		return nil
	}
	dept := &model.DepartmentEntity{
		TenantID: tenant.ID,
		Name:     tenant.Name,
		Code:     tenant.Code,
	}
	if err := db.WithContext(ctx).Create(dept).Error; err != nil {
		return fmt.Errorf("seed root department create fail: %w", err)
	}
	return nil
}

func getOrCreateApplication(ctx context.Context, db *gorm.DB, code, name, desc string, sort int, isSystem int8) (*model.ApplicationEntity, error) {
	entity := &model.ApplicationEntity{}
	err := db.Where("code = ?", code).First(entity).Error
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed application %s query fail: %w", code, err)
	}
	entity = &model.ApplicationEntity{
		Code:        code,
		Name:        name,
		Description: desc,
		Type:        model.AppTypeFirstParty,
		Status:      model.AppStatusEnable,
		Sort:        sort,
		IsSystem:    isSystem,
	}
	if err := db.WithContext(ctx).Create(entity).Error; err != nil {
		return nil, fmt.Errorf("seed application %s create fail: %w", code, err)
	}
	glog.Infof(ctx, "[seed] application created, id:%s code:%s", entity.ID, code)
	return entity, nil
}

func seedRoles(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, app *model.ApplicationEntity) (map[string]*model.RoleEntity, error) {
	defs := []seedRole{
		{code: "admin", name: "管理员", desc: "系统管理员，拥有所有权限", isDefault: 1},
		{code: "user", name: "普通用户", desc: "普通用户，拥有基本查看权限", isDefault: 1},
		{code: "guest", name: "访客", desc: "访客，仅有只读权限", isDefault: 1},
	}
	out := make(map[string]*model.RoleEntity, len(defs))
	for _, def := range defs {
		entity := &model.RoleEntity{}
		err := db.Where("tenant_id = ? AND code = ?", tenant.ID, def.code).First(entity).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("seed role %s query fail: %w", def.code, err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entity = &model.RoleEntity{
				TenantID:    tenant.ID,
				AppID:       app.ID,
				Name:        def.name,
				Code:        def.code,
				Description: def.desc,
				Type:        "User",
				IsDefault:   def.isDefault,
			}
			if err := db.WithContext(ctx).Create(entity).Error; err != nil {
				return nil, fmt.Errorf("seed role %s create fail: %w", def.code, err)
			}
		}
		out[def.code] = entity
	}
	return out, nil
}

func seedResources(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity) (map[string]*model.ResourceEntity, error) {
	defs := []struct {
		indicator string
		name      string
	}{
		{indicator: resourceIndicatorAdmin, name: "管理后台"},
		{indicator: resourceIndicatorMe, name: "用户中心"},
	}
	out := make(map[string]*model.ResourceEntity, len(defs))
	for _, def := range defs {
		entity := &model.ResourceEntity{}
		err := db.Where("tenant_id = ? AND indicator = ?", tenant.ID, def.indicator).First(entity).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("seed resource %s query fail: %w", def.indicator, err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entity = &model.ResourceEntity{
				TenantID:       tenant.ID,
				Name:           def.name,
				Indicator:      def.indicator,
				IsDefault:      1,
				AccessTokenTtl: 3600,
			}
			if err := db.WithContext(ctx).Create(entity).Error; err != nil {
				return nil, fmt.Errorf("seed resource %s create fail: %w", def.indicator, err)
			}
		}
		out[def.indicator] = entity
	}
	return out, nil
}

func seedScopes(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, resources map[string]*model.ResourceEntity) (map[string]*model.ScopeEntity, error) {
	defs := []seedScope{
		{name: "admin:user:read", description: "查看用户", resource: resourceIndicatorAdmin},
		{name: "admin:user:write", description: "管理用户", resource: resourceIndicatorAdmin},
		{name: "admin:role:read", description: "查看角色", resource: resourceIndicatorAdmin},
		{name: "admin:role:write", description: "管理角色", resource: resourceIndicatorAdmin},
		{name: "admin:menu:read", description: "查看菜单", resource: resourceIndicatorAdmin},
		{name: "admin:menu:write", description: "管理菜单", resource: resourceIndicatorAdmin},
		{name: "admin:department:read", description: "查看部门", resource: resourceIndicatorAdmin},
		{name: "admin:department:write", description: "管理部门", resource: resourceIndicatorAdmin},
		{name: "admin:application:read", description: "查看应用", resource: resourceIndicatorAdmin},
		{name: "admin:application:write", description: "管理应用", resource: resourceIndicatorAdmin},
		{name: "admin:resource:read", description: "查看资源", resource: resourceIndicatorAdmin},
		{name: "admin:resource:write", description: "管理资源", resource: resourceIndicatorAdmin},
		{name: "me:profile:read", description: "查看个人信息", resource: resourceIndicatorMe},
		{name: "me:profile:write", description: "修改个人信息", resource: resourceIndicatorMe},
	}
	out := make(map[string]*model.ScopeEntity, len(defs))
	for _, def := range defs {
		res := resources[def.resource]
		entity := &model.ScopeEntity{}
		err := db.Where("tenant_id = ? AND name = ?", tenant.ID, def.name).First(entity).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("seed scope %s query fail: %w", def.name, err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entity = &model.ScopeEntity{
				TenantID:    tenant.ID,
				ResourceID:  res.ID,
				Name:        def.name,
				Description: def.description,
			}
			if err := db.WithContext(ctx).Create(entity).Error; err != nil {
				return nil, fmt.Errorf("seed scope %s create fail: %w", def.name, err)
			}
		}
		out[def.name] = entity
	}
	return out, nil
}

func seedRoleScopes(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, roles map[string]*model.RoleEntity, scopes map[string]*model.ScopeEntity) error {
	// 管理员拥有全部权限
	adminScopes := []string{
		"admin:user:read", "admin:user:write", "admin:role:read", "admin:role:write",
		"admin:menu:read", "admin:menu:write", "admin:department:read", "admin:department:write",
		"admin:application:read", "admin:application:write", "admin:resource:read", "admin:resource:write",
		"me:profile:read", "me:profile:write",
	}
	// 普通用户仅用户中心
	userScopes := []string{"me:profile:read", "me:profile:write"}
	// 访客仅各模块查看权限
	guestScopes := []string{
		"admin:user:read", "admin:role:read", "admin:menu:read", "admin:department:read",
		"admin:application:read", "admin:resource:read", "me:profile:read",
	}
	relations := []struct {
		roleCode   string
		scopeNames []string
	}{
		{roleCode: "admin", scopeNames: adminScopes},
		{roleCode: "user", scopeNames: userScopes},
		{roleCode: "guest", scopeNames: guestScopes},
	}
	for _, rel := range relations {
		role := roles[rel.roleCode]
		for _, scopeName := range rel.scopeNames {
			scope := scopes[scopeName]
			var count int64
			if err := db.Model(&model.RoleScopeEntity{}).
				Where("tenant_id = ? AND role_id = ? AND scope_id = ?", tenant.ID, role.ID, scope.ID).
				Count(&count).Error; err != nil {
				return fmt.Errorf("seed role_scope count fail: %w", err)
			}
			if count > 0 {
				continue
			}
			rs := &model.RoleScopeEntity{TenantID: tenant.ID, RoleID: role.ID, ScopeID: scope.ID}
			if err := db.WithContext(ctx).Create(rs).Error; err != nil {
				return fmt.Errorf("seed role_scope create fail: %w", err)
			}
		}
	}
	return nil
}

func seedMenus(ctx context.Context, db *gorm.DB, adminApp, tenantAdminApp *model.ApplicationEntity) (map[string]*model.MenuEntity, error) {
	defs := []seedMenu{
		// 管理后台一级菜单
		{appCode: appCodeAdmin, name: "工作台", code: "dashboard", path: "/dashboard", icon: "dashboard", sort: 1, component: "Layout", permission: ""},
		{appCode: appCodeAdmin, name: "用户管理", code: "user", path: "/user", icon: "user", sort: 2, component: "Layout", permission: "admin:user:read"},
		{appCode: appCodeAdmin, name: "角色管理", code: "role", path: "/role", icon: "role", sort: 3, component: "Layout", permission: "admin:role:read"},
		{appCode: appCodeAdmin, name: "菜单管理", code: "menu", path: "/menu", icon: "menu", sort: 4, component: "Layout", permission: "admin:menu:read"},
		{appCode: appCodeAdmin, name: "部门管理", code: "department", path: "/department", icon: "department", sort: 5, component: "Layout", permission: "admin:department:read"},
		{appCode: appCodeAdmin, name: "应用管理", code: "application", path: "/application", icon: "app", sort: 6, component: "Layout", permission: "admin:application:read"},
		{appCode: appCodeAdmin, name: "资源管理", code: "resource", path: "/resource", icon: "resource", sort: 7, component: "Layout", permission: "admin:resource:read"},
		// 管理后台二级菜单
		{appCode: appCodeAdmin, parentCode: "user", name: "用户列表", code: "user-list", path: "/user/list", sort: 1, component: "/user/list/index", permission: "admin:user:read"},
		{appCode: appCodeAdmin, parentCode: "role", name: "角色列表", code: "role-list", path: "/role/list", sort: 1, component: "/role/list/index", permission: "admin:role:read"},
		{appCode: appCodeAdmin, parentCode: "role", name: "权限配置", code: "role-permission", path: "/role/permission", sort: 2, component: "/role/permission/index", permission: "admin:role:write"},
		{appCode: appCodeAdmin, parentCode: "menu", name: "菜单列表", code: "menu-list", path: "/menu/list", sort: 1, component: "/menu/list/index", permission: "admin:menu:read"},
		{appCode: appCodeAdmin, parentCode: "department", name: "部门列表", code: "department-list", path: "/department/list", sort: 1, component: "/department/list/index", permission: "admin:department:read"},
		{appCode: appCodeAdmin, parentCode: "application", name: "应用列表", code: "application-list", path: "/application/list", sort: 1, component: "/application/list/index", permission: "admin:application:read"},
		{appCode: appCodeAdmin, parentCode: "resource", name: "资源列表", code: "resource-list", path: "/resource/list", sort: 1, component: "/resource/list/index", permission: "admin:resource:read"},
		// 租户自服务一级菜单
		{appCode: appCodeTenantAdmin, name: "组织管理", code: "organization", path: "/organization", icon: "apartment", sort: 1, component: "pages/organization", permission: ""},
		{appCode: appCodeTenantAdmin, name: "组织角色", code: "organizationRole", path: "/organizationRole", icon: "safety", sort: 2, component: "pages/organizationRole", permission: ""},
		{appCode: appCodeTenantAdmin, name: "组织用户", code: "organizationUser", path: "/organizationUser", icon: "team", sort: 3, component: "pages/organizationUser", permission: ""},
		{appCode: appCodeTenantAdmin, name: "组织角色用户", code: "organizationRoleUser", path: "/organizationRoleUser", icon: "user-switch", sort: 4, component: "pages/organizationRoleUser", permission: ""},
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			parentID := ""
			if def.parentCode != "" {
				parent, ok := out[def.parentCode]
				if !ok {
					return nil, fmt.Errorf("seed menu %s parent %s not found", def.code, def.parentCode)
				}
				parentID = parent.ID
			}
			entity = &model.MenuEntity{
				AppID:      app.ID,
				ParentID:   parentID,
				Name:       def.name,
				Code:       def.code,
				Path:       def.path,
				Icon:       def.icon,
				Sort:       def.sort,
				Type:       menuTypeMenu,
				Component:  def.component,
				Permission: def.permission,
				Status:     menuStatusEnable,
			}
			if err := db.WithContext(ctx).Create(entity).Error; err != nil {
				return nil, fmt.Errorf("seed menu %s create fail: %w", def.code, err)
			}
		}
		out[def.code] = entity
	}
	return out, nil
}

func seedRoleMenus(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, roles map[string]*model.RoleEntity, menus map[string]*model.MenuEntity) error {
	relations := []struct {
		roleCode string
		menuCode []string
	}{
		{roleCode: "admin", menuCode: []string{
			"dashboard", "user", "role", "menu", "department", "application", "resource",
			"user-list", "role-list", "role-permission", "menu-list", "department-list", "application-list", "resource-list",
		}},
		{roleCode: "user", menuCode: []string{"dashboard", "user", "user-list"}},
		{roleCode: "guest", menuCode: []string{"dashboard"}},
	}
	for _, rel := range relations {
		role := roles[rel.roleCode]
		for _, menuCode := range rel.menuCode {
			menu := menus[menuCode]
			var count int64
			if err := db.Model(&model.RoleMenuEntity{}).
				Where("tenant_id = ? AND role_id = ? AND menu_id = ?", tenant.ID, role.ID, menu.ID).
				Count(&count).Error; err != nil {
				return fmt.Errorf("seed role_menu count fail: %w", err)
			}
			if count > 0 {
				continue
			}
			rm := &model.RoleMenuEntity{TenantID: tenant.ID, RoleID: role.ID, MenuID: menu.ID}
			if err := db.WithContext(ctx).Create(rm).Error; err != nil {
				return fmt.Errorf("seed role_menu create fail: %w", err)
			}
		}
	}
	return nil
}

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
		ta := &model.TenantApplicationEntity{TenantID: tenant.ID, AppID: app.ID, Status: model.AppStatusEnable, Config: []byte(`{}`), GrantedScope: []byte(`[]`)}
		if err := db.WithContext(ctx).Create(ta).Error; err != nil {
			return fmt.Errorf("seed tenant_application create fail: %w", err)
		}
	}
	return nil
}

func seedAdminUser(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity) (*model.UserEntity, error) {
	passwordHash, err := gcrypto.GeneratePasswordHash(adminPassword)
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
			PasswordMethod:    "bcrypt",
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
		user = &model.UserEntity{
			TenantID:    tenant.ID,
			PersonID:    person.ID,
			Name:        "系统管理员",
			Profile:     []byte(`{}`),
			CustomData:  []byte(`{}`),
			IsOwner:     1,
			IsSuspended: 0,
		}
		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			return nil, fmt.Errorf("seed admin user create fail: %w", err)
		}
		glog.Infof(ctx, "[seed] admin user created, id:%s (default password: %s)", user.ID, adminPassword)
	}
	return user, nil
}

func seedAdminUserRole(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, adminUser *model.UserEntity, roles map[string]*model.RoleEntity) error {
	adminRole := roles["admin"]
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

func seedOIDCClients(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, app *model.ApplicationEntity) error {
	type clientDef struct {
		clientID             string
		name                 string
		redirectURIs         string
		postLogoutRedirect   string
		backChannelLogoutURI string
	}
	defs := []clientDef{
		{
			clientID:             oauthClientPlatformAdminWeb,
			name:                 "IAM管理平台",
			redirectURIs:         `["http://localhost:3001/auth/callback"]`,
			postLogoutRedirect:   `["http://localhost:3001/login"]`,
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/platform",
		},
		{
			clientID:             oauthClientTenantAdminWeb,
			name:                 "租户管理平台",
			redirectURIs:         `["http://localhost:3002/auth/callback"]`,
			postLogoutRedirect:   `["http://localhost:3002/login"]`,
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/tenant",
		},
	}
	for _, def := range defs {
		entity := &model.ApplicationClientEntity{}
		err := db.Where("client_id = ?", def.clientID).First(entity).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("seed oauth client %s query fail: %w", def.clientID, err)
		}
		entity = &model.ApplicationClientEntity{
			TenantID:                tenant.ID,
			AppID:                   app.ID,
			ClientID:                def.clientID,
			Name:                    def.name,
			RedirectURIs:            []byte(def.redirectURIs),
			PostLogoutRedirectURIs:  []byte(def.postLogoutRedirect),
			BackChannelLogoutURI:    def.backChannelLogoutURI,
			GrantTypes:              []byte(`["authorization_code","refresh_token"]`),
			ResponseTypes:           []byte(`["code"]`),
			TokenEndpointAuthMethod: "none",
			RequirePKCE:             1,
			DefaultScopes:           []byte(`["openid","profile","email"]`),
			Type:                    model.AppTypeFirstParty,
			Status:                  model.AppStatusEnable,
			IsSystem:                1,
		}
		if err := db.WithContext(ctx).Create(entity).Error; err != nil {
			return fmt.Errorf("seed oauth client %s create fail: %w", def.clientID, err)
		}
		glog.Infof(ctx, "[seed] oauth client created, clientID:%s", def.clientID)
	}
	return nil
}
