// Package seed 提供 IAM 基础种子数据的幂等写入能力。
//
// 替代历史 MySQL 方言建表/种子脚本（scripts/sql/*.sql 已废弃删除）：服务启动时
// 基于唯一键（code / client_id / username 等）查重，已存在则跳过、不存在则创建，
// 因此可安全重复执行，兼容全新数据库与已有数据的升级场景。
// 自 string-id 改造起所有主键为字符串（UUID v7），实体间关联在写入时动态接线，
// 不再依赖固定的数字主键。
//
// 业务约束：用户必须从属于某个部门（组织节点），种子管理员同样从属于
// 租户的顶级部门（根组织节点，seedRootOrganization 创建），归属关系为
// member 行政主部门。
package seed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

const (
	tenantCodePlatform = "platform"

	adminPassword = "admin123"

	appCodeAdmin       = "platform-admin"
	appCodeTenantAdmin = "tenant-admin"

	oauthClientPlatformAdminWeb = "platform-admin-web"
	oauthClientTenantAdminWeb   = "tenant-admin-web"
)

// seedRole 角色种子定义。
type seedRole struct {
	code       string
	name       string
	desc       string
	adminLevel string // 系统管理等级(member/super)，空=无系统管理能力
}

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
	visibility model.MenuVisibility
}

// SeedIam 幂等写入 IAM 基础种子数据。任一环节失败即返回错误，由调用方决定是否阻断启动。
func SeedIam(ctx context.Context, db *gorm.DB) error {
	// 1. 平台租户
	tenant, err := getOrCreateTenant(ctx, db)
	if err != nil {
		return err
	}

	// 2. 租户同名顶级部门（用户归属的根组织，管理员也归属于此）
	rootOrg, err := seedRootOrganization(ctx, db, tenant)
	if err != nil {
		return err
	}

	// 3. 应用
	adminApp, err := getOrCreateApplication(ctx, db, appCodeAdmin, "管理后台", "平台管理后台应用", 0, true)
	if err != nil {
		return err
	}
	tenantAdminApp, err := getOrCreateApplication(ctx, db, appCodeTenantAdmin, "租户自服务", "租户自服务控制台应用", 1, false)
	if err != nil {
		return err
	}

	// 4. 角色
	roles, err := seedRoles(ctx, db, tenant, adminApp)
	if err != nil {
		return err
	}

	// 5. 菜单
	menus, err := seedMenus(ctx, db, adminApp, tenantAdminApp)
	if err != nil {
		return err
	}

	// 6. 角色-菜单关联
	if err := seedRoleMenus(ctx, db, tenant, roles, menus); err != nil {
		return err
	}

	// 7. 租户应用订阅
	if err := seedTenantApplications(ctx, db, tenant, adminApp, tenantAdminApp); err != nil {
		return err
	}

	// 8. 默认管理员（person + user + user_role + 顶级部门归属）
	adminUser, err := seedAdminUser(ctx, db, tenant, rootOrg)
	if err != nil {
		return err
	}
	if err := seedAdminUserRole(ctx, db, tenant, adminUser, roles); err != nil {
		return err
	}

	// 9. OIDC 测试客户端
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
		IsSuspended: false,
		Tag:         "default",
	}
	if err := db.WithContext(ctx).Create(entity).Error; err != nil {
		return nil, fmt.Errorf("seed tenant create fail: %w", err)
	}
	glog.Infof(ctx, "[seed] tenant created, id:%s code:%s", entity.ID, entity.Code)
	return entity, nil
}

// seedRootOrganization 确保租户存在唯一顶级部门（根组织节点），并返回该节点。
// 所有种子用户（含管理员）均从属于此顶级部门。
func seedRootOrganization(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity) (*model.OrganizationEntity, error) {
	org := &model.OrganizationEntity{}
	err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(org).Error
	if err == nil {
		return org, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed root organization query fail: %w", err)
	}
	org = &model.OrganizationEntity{
		TenantID: tenant.ID,
		Name:     tenant.Name,
		Code:     tenant.Code,
		Status:   string(model.OrgNodeStatusActive),
	}
	if err := db.WithContext(ctx).Create(org).Error; err != nil {
		return nil, fmt.Errorf("seed root organization create fail: %w", err)
	}
	// 根节点路径："/"+id，深度 1（ID 由 BeforeCreate 生成，需创建后补写）
	if err := db.WithContext(ctx).Model(org).Updates(map[string]any{
		"org_path":  "/" + org.ID,
		"org_depth": 1,
	}).Error; err != nil {
		return nil, fmt.Errorf("seed root organization path fail: %w", err)
	}
	return org, nil
}

func getOrCreateApplication(ctx context.Context, db *gorm.DB, code, name, desc string, sort int, isSystem bool) (*model.ApplicationEntity, error) {
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
		{code: "admin", name: "管理员", desc: "系统管理员，拥有所有权限", adminLevel: string(model.SysAdminLevelSuper)},
	}
	out := make(map[string]*model.RoleEntity, len(defs))
	for _, def := range defs {
		adminLevel := def.adminLevel
		if adminLevel == "" {
			adminLevel = string(model.SysAdminLevelMember)
		}
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
				Source:      string(model.RoleSourceBuiltin),
				AdminLevel:  adminLevel,
			}
			if err := db.WithContext(ctx).Create(entity).Error; err != nil {
				return nil, fmt.Errorf("seed role %s create fail: %w", def.code, err)
			}
		} else {
			// 幂等回填：存量内置角色的 source / admin_level 随种子定义更新
			updateMap := map[string]any{}
			if entity.Source != string(model.RoleSourceBuiltin) {
				updateMap["source"] = string(model.RoleSourceBuiltin)
			}
			if entity.AdminLevel != adminLevel {
				updateMap["admin_level"] = adminLevel
			}
			if len(updateMap) > 0 {
				if uerr := db.Model(&model.RoleEntity{}).Where("id = ?", entity.ID).Updates(updateMap).Error; uerr != nil {
					return nil, fmt.Errorf("seed role %s update fail: %w", def.code, uerr)
				}
				entity.Source = string(model.RoleSourceBuiltin)
				entity.AdminLevel = adminLevel
			}
		}
		out[def.code] = entity
	}
	return out, nil
}

func seedMenus(ctx context.Context, db *gorm.DB, adminApp, tenantAdminApp *model.ApplicationEntity) (map[string]*model.MenuEntity, error) {
	defs := []seedMenu{
		// 管理后台一级菜单
		{appCode: appCodeAdmin, name: "工作台", code: "dashboard", path: "/dashboard", icon: "dashboard", sort: 1, component: "Layout", visibility: model.MenuVisibilityMember},
		{appCode: appCodeAdmin, name: "用户管理", code: "user", path: "/user", icon: "user", sort: 2, component: "Layout", visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, name: "角色管理", code: "role", path: "/role", icon: "role", sort: 3, component: "Layout", visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, name: "菜单管理", code: "menu", path: "/menu", icon: "menu", sort: 4, component: "Layout", visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, name: "应用管理", code: "application", path: "/application", icon: "app", sort: 6, component: "Layout", visibility: model.MenuVisibilityAdmin},
		// 管理后台二级菜单
		{appCode: appCodeAdmin, parentCode: "user", name: "用户列表", code: "user-list", path: "/user/list", sort: 1, component: "/user/list/index", visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "role", name: "角色列表", code: "role-list", path: "/role/list", sort: 1, component: "/role/list/index", visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "menu", name: "菜单列表", code: "menu-list", path: "/menu/list", sort: 1, component: "/menu/list/index", visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeAdmin, parentCode: "application", name: "应用列表", code: "application-list", path: "/application/list", sort: 1, component: "/application/list/index", visibility: model.MenuVisibilityAdmin},
		// 租户自服务一级菜单（组织架构拆为组织管理/成员管理；用户/角色编码加 tenant- 前缀，避免与平台菜单 code 撞名）
		{appCode: appCodeTenantAdmin, name: "组织管理", code: "organization", path: "/organization", icon: "apartment", sort: 1, component: "pages/organization", visibility: model.MenuVisibilityPublic},
		{appCode: appCodeTenantAdmin, name: "成员管理", code: "organization-member", path: "/organization/members", icon: "team", sort: 2, component: "pages/organization-member", visibility: model.MenuVisibilityPublic},
		{appCode: appCodeTenantAdmin, name: "用户管理", code: "tenant-user", path: "/user", icon: "user", sort: 3, component: "pages/user", visibility: model.MenuVisibilityAdmin},
		{appCode: appCodeTenantAdmin, name: "角色管理", code: "tenant-role", path: "/role", icon: "role", sort: 4, component: "pages/role", visibility: model.MenuVisibilityAdmin},
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
				Type:       model.MenuTypeMenu,
				Visibility: visibility,
				Component:  def.component,
				Status:     model.MenuStatusEnable,
			}
			if err := db.WithContext(ctx).Create(entity).Error; err != nil {
				return nil, fmt.Errorf("seed menu %s create fail: %w", def.code, err)
			}
		} else if entity.Visibility != visibility {
			// 幂等回填：存量菜单的可见性门槛随种子定义更新，避免沿用旧默认值
			if uerr := db.Model(&model.MenuEntity{}).Where("id = ?", entity.ID).
				UpdateColumn("visibility", visibility).Error; uerr != nil {
				return nil, fmt.Errorf("seed menu %s update visibility fail: %w", def.code, uerr)
			}
			entity.Visibility = visibility
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
			"dashboard", "user", "role", "menu", "application",
			"user-list", "role-list", "menu-list", "application-list",
			"organization", "organization-member", "tenant-user", "tenant-role",
		}},
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

// seedAdminUser 幂等写入默认管理员（person + user），并确保其从属于顶级部门 rootOrg
// （primary 行政主部门），满足"用户必须从属于某个部门"的业务约束。
// rootOrg 缺失时视为种子数据不完整，直接报错，避免产出无归属用户。
func seedAdminUser(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, rootOrg *model.OrganizationEntity) (*model.UserEntity, error) {
	if rootOrg == nil || rootOrg.ID == "" {
		return nil, fmt.Errorf("seed admin user fail: root organization not found")
	}
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
		now := time.Now()
		user = &model.UserEntity{
			TenantID:    tenant.ID,
			PersonID:    person.ID,
			Name:        "系统管理员",
			Profile:     []byte(`{}`),
			CustomData:  []byte(`{}`),
			IsOwner:     true,
			IsSuspended: false,
			JoinedAt:    &now,
		}
		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			return nil, fmt.Errorf("seed admin user create fail: %w", err)
		}
		glog.Infof(ctx, "[seed] admin user created, id:%s (default password: %s)", user.ID, adminPassword)
	}

	// 顶级部门归属（幂等，兼容已有库升级：admin 用户已存在但尚无组织归属的场景）
	if err := seedAdminUserOrganization(ctx, db, tenant, user, rootOrg); err != nil {
		return nil, err
	}
	return user, nil
}

// seedAdminUserOrganization 幂等建立管理员与顶级部门的行政主部门关系（primary）。
// 该函数独立于用户创建之外执行，保证升级场景（用户已存在、归属缺失）也能补齐。
func seedAdminUserOrganization(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, user *model.UserEntity, rootOrg *model.OrganizationEntity) error {
	var count int64
	if err := db.Model(&model.OrganizationUserEntity{}).
		Where("tenant_id = ? AND user_id = ? AND organization_id = ? AND relation_type = ?",
			tenant.ID, user.ID, rootOrg.ID, model.OrgUserRelationPrimary).
		Count(&count).Error; err != nil {
		return fmt.Errorf("seed admin organization count fail: %w", err)
	}
	if count > 0 {
		return nil
	}
	relation := &model.OrganizationUserEntity{
		TenantID:       tenant.ID,
		OrganizationID: rootOrg.ID,
		UserID:         user.ID,
		RelationType:   model.OrgUserRelationPrimary,
	}
	if err := db.WithContext(ctx).Create(relation).Error; err != nil {
		return fmt.Errorf("seed admin organization create fail: %w", err)
	}
	glog.Infof(ctx, "[seed] admin organization relation created, user_id:%s org_id:%s", user.ID, rootOrg.ID)
	return nil
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
			GrantTypes:              []byte(`["authorization_code","refresh_token"]`),
			ResponseTypes:           []byte(`["code"]`),
			TokenEndpointAuthMethod: "none",
			RequirePKCE:             true,
			DefaultScopes:           []byte(`["openid","profile","email"]`),
			Type:                    model.AppTypeFirstParty,
			Status:                  model.AppStatusEnable,
			IsSystem:                true,
		}
		if err := db.WithContext(ctx).Create(entity).Error; err != nil {
			return fmt.Errorf("seed oauth client %s create fail: %w", def.code, err)
		}
		glog.Infof(ctx, "[seed] oauth client created, code:%s", def.code)
	}
	return nil
}
