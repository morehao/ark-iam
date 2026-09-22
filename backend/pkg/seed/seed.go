// Package seed 提供 IAM 内置数据的一次性引导能力（L1 Bootstrap）。
//
// 唯一触发者是初始化页面（POST /install/initialize）；**启动期不做任何数据写入**，
// 只执行 AutoMigrate 建表（见 pkg/model.AutoMigrateAll）。库是否已初始化由平台租户行
// 判定（IsInitialized），已初始化后 Bootstrap 永久自锁，不再写任何数据。
//
// 本包因此只有一条写通道（Bootstrap）：按不可见的种子身份键 seed_key 认行、缺失则创建。
// 内置行的**后续调整一律归运维**（各控制台页面）：本包不再做跨版本的字段收敛、改名或
// 退役清理——那些机制（reconcileFields / seedMigrations / retiredMenus / 墓碑跳过）
// 已随「内置数据交给运维」一并删除，详见 docs/design/system-design.md §4.5。
// 新增版本菜单由 `make print-builtin-menus` 输出清单、运维在「菜单管理」页补录。
//
// 字段权威矩阵（pkg/model.SeedFieldAuthorities）保留为**声明 + 契约测试**：它说明哪些
// 字段在 L1 创建时写入、之后控制台必须拒写，但执行体是各 service 的拒写点，不是本包。
//
// 自 string-id 改造起所有主键为字符串（UUID v7），实体间关联在写入时动态接线，
// 不再依赖固定的数字主键。
//
// 业务约束：用户必须从属于某个部门（部门节点），引导管理员同样从属于
// 租户的顶级部门（根部门节点，seedRootDepartment 创建），归属关系为 primary 行政主部门。
package seed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	// 别名：SeedIam 内以 tenant 命名的局部变量会遮蔽同名包
	iamtenant "github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

const (
	// tenantCodePlatform 平台租户（平台运营中心）编码：取自字段权威矩阵所在包（pkg/model），
	// 控制台"平台租户不可挂起"的判定与本处的定位共用同一份常量。形态与自动生成规则
	// （pkg/core/tenant.GenerateCode：t_<12 位随机 hex>）一致——同前缀、后缀可读且固定。
	// 自动生成编码的随机段只用小写 hex，"platform" 含非 hex 字符，故两者永不冲突。
	tenantCodePlatform = model.SeedPlatformTenantCode
	// changeActionCreated Report 里唯一的动作：L1 只创建，不更新、不改名。
	changeActionCreated = "created"

	// 平台租户名（defaultTenantName）在 definition.go：它是 L1 的输入（Definition.TenantName），
	// 不再是包内常量；此处保留说明，避免读者按旧路径查找。

	// seedAdvisoryLockKey 播种互斥键（Postgres 事务级 advisory lock）：分体部署时四个应用会同时
	// 启动播种，用它把执行串行化，避免并发插入撞关联表唯一索引后中断启动。取值仅需全系统一致。
	seedAdvisoryLockKey = int64(0x61726b5f69616d)

	// 应用编码规则：小写字母开头，仅含小写字母/数字/下划线（model.AppCodePattern），
	// 与自动生成的租户编码（t_<hex>）同为下划线连接。
	appCodeAdmin       = "platform_admin"
	appCodeTenantAdmin = "tenant_admin"

	// 内置 OAuth 客户端编码（= OIDC client_id）取自 pkg/model 的种子身份常量：
	// 网关侧用它做令牌 audience 校验，两处必须是同一个值。
	oauthClientPlatformAdminWeb = model.SeedBuiltinClientPlatformAdminWeb
	oauthClientTenantAdminWeb   = model.SeedBuiltinClientTenantAdminWeb
)

// Change 一次 L1 写入：Action 恒为 created（L1 只创建，不更新、不改名）。
type Change struct {
	Entity string
	Key    string
	Action string
}

// Report 本次种子执行的变更报告：仅用于启动日志核对（不落库），
// 覆盖权威矩阵相关实体与平台自举产物（租户/部门/应用/菜单/客户端/角色/订阅/管理员）。
type Report struct {
	// TenantID 平台租户 ID：L1 引导的审计与响应需要它定位本次操作的目标租户。
	// 引导失败（未走到租户创建）时为空。
	TenantID string
	Changes  []Change
}

func (r *Report) created(entity, key string) {
	r.Changes = append(r.Changes, Change{Entity: entity, Key: key, Action: changeActionCreated})
}

// Summary 汇总变更计数（启动日志一行）。L1 只创建，故仅有 created 一项。
func (r *Report) Summary() string {
	return fmt.Sprintf("created=%d", len(r.Changes))
}

// log 输出变更报告：汇总一行，便于部署时核对"这次启动写了什么"。
func (r *Report) log(ctx context.Context) {
	if len(r.Changes) == 0 {
		return
	}
	glog.Infof(ctx, "[seed] done, %s", r.Summary())
}

// lockSeed 播种互斥：Postgres 取事务级 advisory lock（事务结束自动释放）。
// SQLite 等测试库没有 advisory lock，单进程测试无并发风险，直接跳过。
func lockSeed(tx *gorm.DB) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", seedAdvisoryLockKey).Error; err != nil {
		return fmt.Errorf("seed advisory lock fail: %w", err)
	}
	return nil
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
	menuType   model.MenuType
	visibility model.MenuVisibility
}

// bootstrapAll 顺序执行 L1 各类引导（在 Bootstrap 的单个事务内）。步骤编号与依赖顺序一一对应。
func bootstrapAll(ctx context.Context, db *gorm.DB, rep *Report, def Definition) error {
	// 1. 平台租户
	tenant, err := getOrCreateTenant(ctx, db, rep, def)
	if err != nil {
		return err
	}
	// 审计与响应据此定位本次引导的目标租户
	rep.TenantID = tenant.ID

	// 2. 租户同名顶级部门（用户归属的根部门，管理员也归属于此）
	rootDept, err := seedRootDepartment(ctx, db, tenant, rep)
	if err != nil {
		return err
	}

	// 3. 应用
	adminApp, err := getOrCreateApplication(ctx, db, rep, appCodeAdmin, "平台管理后台", "平台管理后台应用", 0, model.AppSourceBuiltin)
	if err != nil {
		return err
	}
	tenantAdminApp, err := getOrCreateApplication(ctx, db, rep, appCodeTenantAdmin, "租户管理后台", "租户管理后台应用", 1, model.AppSourceBuiltin)
	if err != nil {
		return err
	}

	// 4. 角色（admin 归属平台管理后台；tenant_admin 由第 9 步的权限开通统一创建）
	adminRole, err := seedRoles(ctx, db, rep, tenant, adminApp)
	if err != nil {
		return err
	}

	// 5. 菜单
	menus, err := seedMenus(ctx, db, rep, adminApp, tenantAdminApp)
	if err != nil {
		return err
	}

	// 6. 角色-菜单关联（仅平台管理后台 admin 角色；tenant_admin 由第 9 步开通时授权）
	if err := seedRoleMenus(ctx, db, tenant, adminRole, menus); err != nil {
		return err
	}

	// 7. 租户应用订阅（平台管理后台 platform_admin；租户管理后台 tenant_admin 由第 9 步开通时订阅）
	if err := seedTenantApplications(ctx, db, rep, tenant, adminApp); err != nil {
		return err
	}

	// 8. 默认管理员（person + user + 顶级部门归属）
	adminUser, err := seedAdminUser(ctx, db, rep, tenant, rootDept, def)
	if err != nil {
		return err
	}
	if err := seedAdminUserRole(ctx, db, rep, tenant, adminUser, adminRole); err != nil {
		return err
	}

	// 9. 平台租户的租户自服务权限开通：与"新建租户"共用同一实现
	// （pkg/core/tenant.ProvisionTenantAdmin），保证内置角色/菜单授权/订阅只有一份定义。
	if _, err := iamtenant.ProvisionTenantAdmin(ctx, db, &iamtenant.ProvisionTenantAdminReq{
		TenantID:    tenant.ID,
		GrantUserID: adminUser.ID,
	}); err != nil {
		return fmt.Errorf("seed provision tenant admin fail: %w", err)
	}

	// 10. OIDC 测试客户端（平台管理后台客户端挂 platform_admin，租户管理后台客户端挂 tenant_admin）
	if err := seedOIDCClients(ctx, db, rep, tenant, adminApp, tenantAdminApp, def); err != nil {
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

func getOrCreateTenant(ctx context.Context, db *gorm.DB, rep *Report, def Definition) (*model.TenantEntity, error) {
	entity, err := findTenantByCode(db, tenantCodePlatform)
	if err != nil {
		return nil, err
	}
	if entity != nil {
		// 已存在即返回：平台租户的 name/status/type/tag/db_user 全归运维（create_only 语义），
		// 引导不回写任何字段。平台租户被挂起会导致整栈控制台失联，这条不变式改由控制台的
		// 拒写点保证（平台租户不可挂起），不再靠"每次启动收敛 status"来兜底。
		return entity, nil
	}
	entity = &model.TenantEntity{
		Code:   tenantCodePlatform,
		Name:   def.TenantName,
		Type:   model.TenantTypePlatform,
		DbUser: "default_user",
		Status: model.TenantStatusActive,
		Tag:    "default",
	}
	if err := db.WithContext(ctx).Create(entity).Error; err != nil {
		return nil, fmt.Errorf("seed tenant create fail: %w", err)
	}
	glog.Infof(ctx, "[seed] tenant created, id:%s code:%s", entity.ID, entity.Code)
	rep.created(model.SeedEntityTenant, entity.Code)
	return entity, nil
}

// seedRootDepartment 确保租户存在唯一顶级部门（根部门节点），并返回该节点。
// 根部门名与租户同名只是**创建时**的取值：运维按自己组织架构改过部门名后引导不回写
// （create_only 语义）。所有种子用户（含管理员）均从属于此顶级部门。
func seedRootDepartment(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, rep *Report) (*model.DepartmentEntity, error) {
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
	rep.created(model.SeedEntityDepartment, dept.ID)
	// 根节点路径："/"+id，深度 1（ID 由 BeforeCreate 生成，需创建后补写）
	if err := db.WithContext(ctx).Model(dept).Updates(map[string]any{
		"dept_path":  "/" + dept.ID,
		"dept_depth": 1,
	}).Error; err != nil {
		return nil, fmt.Errorf("seed root department path fail: %w", err)
	}
	return dept, nil
}

// findApplicationBySeedKey 按种子身份键查内置应用；不存在返回 (nil, nil)，系统错误返回 (nil, err)。
func findApplicationBySeedKey(db *gorm.DB, seedKey string) (*model.ApplicationEntity, error) {
	entity := &model.ApplicationEntity{}
	err := db.Where("seed_key = ?", seedKey).First(entity).Error
	if err == nil {
		return entity, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("seed application query fail (seed_key=%s): %w", seedKey, err)
}

// getOrCreateApplication 幂等获取内置应用：按不可见的种子身份键 seed_key 认行，缺失则创建。
//
// 只按 seed_key 认行，不做 (code) 兜底：本项目按全新项目维护 schema，不存在"还没有 seed_key
// 的存量行"；保留旧口径兜底反而会让运维自建的同 code 应用被误认领成内置行（获得删除保护、
// 接管菜单范围）。编码改名不影响任何以 app_id 关联的菜单/订阅/角色。
func getOrCreateApplication(ctx context.Context, db *gorm.DB, rep *Report, code, name, desc string, sort int, source model.AppSource) (*model.ApplicationEntity, error) {
	entity, err := findApplicationBySeedKey(db, code)
	if err != nil {
		return nil, err
	}
	if entity != nil {
		// 已存在即返回：name/description/sort/status 全归运维（create_only 语义），引导不回写
		// 任何字段——运营在控制台改过的名称与描述在重新初始化（不可能发生）或重建库后不会被收回。
		// source（内置标记，安全不变式）由服务端的拒写点保证不可更改，不再靠种子收敛。
		return entity, nil
	}
	entity = &model.ApplicationEntity{
		Code:        code,
		SeedKey:     code,
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
	rep.created(model.SeedEntityApplication, code)
	return entity, nil
}

// seedRoles 只种平台管理后台 admin 角色：租户管理后台的 tenant_admin 由权限开通统一创建
// （pkg/core/tenant.ProvisionTenantAdmin），建租户与种子共用同一份角色/授权定义。
// 角色编码（model.RoleCode）是跨系统授权契约值，两种内置角色的取值都在 pkg/model 集中定义；
// 幂等定位键为 (tenant_id, app_id, source=builtin)，编码按 create_only 语义只在创建时写入。
func seedRoles(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, adminApp *model.ApplicationEntity) (*model.RoleEntity, error) {
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
			Code:        model.RoleCodePlatformAdmin,
			Name:        "管理员",
			Description: "系统管理员，拥有所有权限",
			Source:      model.RoleSourceBuiltin,
			AdminType:   adminType,
		}
		if err := db.WithContext(ctx).Create(entity).Error; err != nil {
			return nil, fmt.Errorf("seed admin role create fail: %w", err)
		}
		rep.created(model.SeedEntityRole, entity.Name)
		return entity, nil
	}
	// 已有角色直接复用，**不做任何回写**——包括 admin_type。
	//
	// 这里曾经有一段"幂等回填：存量内置角色的 admin_type 随种子定义更新"，
	// 属跨版本写入的残留，已删除，理由有两条：
	//   1) 不可达：Bootstrap 只在库未初始化时执行且成功即永久自锁，
	//      走到这个分支意味着库内已存在内置角色，那不是本函数该处理的场景；
	//   2) 语义错误：`role.admin_type` 在字段权威矩阵里是 immutable，
	//      控制台已拒改（见 svctenant 的角色更新校验），因此这里"收敛"不到任何运维改动，
	//      只会让读者以为种子仍具备跨版本写入能力——而这次改造的全部意义就是取消它。
	return entity, nil
}

// menuTypeOf 返回菜单种子定义的 type；未显式指定时缺省为 menu。
func menuTypeOf(def seedMenu) model.MenuType {
	if def.menuType == "" {
		return model.MenuTypeMenu
	}
	return def.menuType
}

func seedMenus(ctx context.Context, db *gorm.DB, rep *Report, adminApp, tenantAdminApp *model.ApplicationEntity) (map[string]*model.MenuEntity, error) {

	appByCode := map[string]*model.ApplicationEntity{appCodeAdmin: adminApp, appCodeTenantAdmin: tenantAdminApp}
	out := make(map[string]*model.MenuEntity, len(builtinMenuDefs))
	for _, def := range builtinMenuDefs {
		app := appByCode[def.appCode]
		// 认行只看不可见的种子身份键 seed_key（见 model.MenuEntity.SeedKey）：运营在控制台改过
		// code、或把菜单换挂到别的应用后仍命中同一行。
		//
		// 控制台删除内置菜单后不会被"建回来"：L1 已初始化即永久自锁（见 Bootstrap），
		// 不存在第二次写入。这正是退役机制（retiredMenus）与墓碑跳过（menuSeedKeyRemoved）
		// 可以删除的原因——它们要解决的是"种子反复执行会撤销运维改动"，而 L1 只执行一次。
		//
		// 菜单的展示与结构字段全部归运维（create_only），命中后不回写任何字段。
		entity, err := findMenuBySeedKey(db, def.code)
		if err != nil {
			return nil, err
		}
		parentID := ""
		if def.parentCode != "" {
			parent, ok := out[def.parentCode]
			if !ok || parent == nil {
				// builtinMenuDefs 保证父级先于子级出现；走到这里说明定义写错了，
				// 属于编程错误而非数据问题，直接中断而不是静默跳过子菜单。
				return nil, fmt.Errorf("seed menu %s 的父级 %s 未在其之前定义", def.code, def.parentCode)
			}
			parentID = parent.ID
		}
		// 与 BuiltinMenus() 共用同一份缺省逻辑，避免"清单显示 public、落库却是别的值"
		visibility := menuVisibilityOf(def)
		if entity == nil {
			entity = &model.MenuEntity{
				AppID:      app.ID,
				SeedKey:    def.code,
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
			glog.Infof(ctx, "[seed] menu created, id:%s code:%s", entity.ID, def.code)
			rep.created(model.SeedEntityMenu, def.code)
		}
		out[def.code] = entity
	}
	return out, nil
}

// findMenuBySeedKey 按种子身份键查内置菜单；不存在返回 (nil, nil)，系统错误返回 (nil, err)。
func findMenuBySeedKey(db *gorm.DB, seedKey string) (*model.MenuEntity, error) {
	entity := &model.MenuEntity{}
	err := db.Where("seed_key = ?", seedKey).First(entity).Error
	if err == nil {
		return entity, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("seed menu query fail (seed_key=%s): %w", seedKey, err)
}

// seedRoleMenus 只处理平台管理后台 admin 角色的菜单授权；
// tenant_admin 的菜单授权由 pkg/core/tenant.ProvisionTenantAdmin 统一写入。
func seedRoleMenus(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, adminRole *model.RoleEntity, menus map[string]*model.MenuEntity) error {
	menuCodes := []string{
		"dashboard", "menu", "tenant", "application",
		"tenant-application", "oauth-client", "domain", "log",
		"department", "tenant-user", "tenant-role",
	}
	for _, menuCode := range menuCodes {
		menu := menus[menuCode]
		if menu == nil || menu.ID == "" {
			// 该内置菜单已被控制台删除（墓碑）：authorization 已随删除清理，不补授权
			continue
		}
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

// seedTenantApplications 写入平台管理后台的应用订阅；租户管理后台（tenant_admin）的订阅
// 由 pkg/core/tenant.ProvisionTenantAdmin 统一写入。
func seedTenantApplications(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, apps ...*model.ApplicationEntity) error {
	for _, app := range apps {
		var count int64
		if err := db.Model(&model.TenantApplicationEntity{}).
			Where("tenant_id = ? AND app_id = ?", tenant.ID, app.ID).Count(&count).Error; err != nil {
			return fmt.Errorf("seed tenant_application count fail: %w", err)
		}
		if count > 0 {
			continue
		}
		ta := &model.TenantApplicationEntity{TenantID: tenant.ID, AppID: app.ID, Status: model.TenantApplicationStatusEnable}
		if err := db.WithContext(ctx).Create(ta).Error; err != nil {
			return fmt.Errorf("seed tenant_application create fail: %w", err)
		}
		rep.created(model.TableNameTenantApplication, app.Code)
	}
	return nil
}

// seedAdminUser 幂等写入默认管理员（person + user），并确保其从属于顶级部门 rootDept
// （primary 行政主部门），满足"用户必须从属于某个部门"的业务约束。
// rootDept 缺失时视为种子数据不完整，直接报错，避免产出无归属用户。
func seedAdminUser(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, rootDept *model.DepartmentEntity, def Definition) (*model.UserEntity, error) {
	if rootDept == nil || rootDept.ID == "" {
		return nil, fmt.Errorf("seed admin user fail: root department not found")
	}
	// person：以 username 为唯一键。口令摘要由调用方提供（Bootstrap 的调用方是初始化接口，
	// 过渡期的 Run 由 legacyDefinition 生成），本包不内置任何默认口令。
	person := &model.PersonEntity{}
	pErr := db.Where("username = ?", model.StrPtr(def.AdminUsername)).First(person).Error
	if pErr != nil && !errors.Is(pErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seed admin person query fail: %w", pErr)
	}
	if errors.Is(pErr, gorm.ErrRecordNotFound) {
		person = &model.PersonEntity{
			Username:          model.StrPtr(def.AdminUsername),
			PrimaryEmail:      model.StrPtr(def.AdminEmail),
			PrimaryPhone:      model.StrPtr(def.AdminPhone),
			PasswordEncrypted: def.AdminPasswordHash,
			PasswordMethod:    model.PasswordMethodBcrypt,
			PasswordStatus:    def.AdminPasswordStatus,
			Name:              def.AdminName,
		}
		if err := db.WithContext(ctx).Create(person).Error; err != nil {
			return nil, fmt.Errorf("seed admin person create fail: %w", err)
		}
		rep.created(model.SeedEntityPerson, def.AdminUsername)
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
			TenantID:  tenant.ID,
			PersonID:  person.ID,
			Name:      def.AdminName,
			Source:    model.UserSourceBuiltin,
			OwnerType: model.OwnerTypeOwner,
			Status:    model.UserStatusActive,
			JoinedAt:  &now,
		}
		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			return nil, fmt.Errorf("seed admin user create fail: %w", err)
		}
		glog.Infof(ctx, "[seed] admin user created, id:%s username:%s password_status:%s", user.ID, def.AdminUsername, def.AdminPasswordStatus)
		rep.created(model.SeedEntityUser, user.ID)
	}

	// source=builtin 在创建时即写入（见上方 Create），不做启动期回填：它是账号归属的
	// 安全不变式，由控制台的拒写点保证不可更改；平台侧"重置内置管理员密码"依赖该标记定位用户。

	// 顶级部门归属（幂等，兼容已有库升级：admin 用户已存在但尚无部门归属的场景）
	if err := seedAdminUserDepartment(ctx, db, rep, tenant, user, rootDept); err != nil {
		return nil, err
	}
	return user, nil
}

// seedAdminUserDepartment 幂等建立管理员与顶级部门的行政主部门关系（primary）。
// 该函数独立于用户创建之外执行，保证升级场景（用户已存在、归属缺失）也能补齐。
func seedAdminUserDepartment(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, user *model.UserEntity, rootDept *model.DepartmentEntity) error {
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
	rep.created(model.TableNameDepartmentUser, relation.ID)
	return nil
}

// seedAdminUserRole 绑定平台管理后台 admin 角色；默认管理员的 tenant_admin 角色绑定
// 由后续的权限开通（pkg/core/tenant.ProvisionTenantAdmin）统一写入。
func seedAdminUserRole(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, adminUser *model.UserEntity, adminRole *model.RoleEntity) error {
	// 默认管理员持有平台管理后台 admin 角色
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
	rep.created(model.TableNameUserRole, ur.ID)
	return nil
}

// seedOIDCClientGrantTypes 种子 OAuth 客户端授权类型：引用 model.GrantType 常量，
// 避免在种子数据里裸写 JSON 字面量导致取值漂移。
var seedOIDCClientGrantTypes = model.GrantTypeList{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken}

// findApplicationClientByCode 按编码查客户端（code 即 OIDC client_id）；
// 不存在返回 (nil, nil)，系统错误返回 (nil, err)。
func findApplicationClientByCode(db *gorm.DB, code string) (*model.ApplicationClientEntity, error) {
	entity := &model.ApplicationClientEntity{}
	err := db.Where("code = ?", code).First(entity).Error
	if err == nil {
		return entity, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("seed application_client query fail (code=%s): %w", code, err)
}

// seedOIDCClients 播种内置 OAuth 客户端（OIDC RP），按 code（= client_id）认行。
// 归属应用按控制台一一对应：平台管理后台客户端 → platform_admin 应用，租户管理后台客户端 → tenant_admin 应用。
// app_id 只在创建时写入（create_only）：控制台拒改内置客户端的 code，但归属应用的调整由控制台完成，
// 引导不回写——否则运维的改动会在下次（不可能发生的）初始化时被收回。
//
// 回调地址来自 def.Consoles（完整 URL，不做 origin 派生）；bc-logout 已由 withDefaults 解析成最终值，
// 因此这里落库的地址与初始化页面回显给运维的地址必然一致。
// 切片非 nil 由 withDefaults 保证（nil → 内置缺省），满足 JSON 列"nil 入库前归一为空切片"的约束。
func seedOIDCClients(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, adminApp, tenantAdminApp *model.ApplicationEntity, def Definition) error {
	type builtinClientDef struct {
		code                 string
		name                 string
		appID                string
		redirectURIs         model.RedirectURIList
		postLogoutRedirect   model.PostLogoutRedirectURIList
		backChannelLogoutURI string
	}
	clientDefs := []builtinClientDef{
		{
			code:                 oauthClientPlatformAdminWeb,
			name:                 "平台管理后台",
			appID:                adminApp.ID,
			redirectURIs:         model.RedirectURIList(def.Consoles.PlatformAdminWeb.RedirectURIs),
			postLogoutRedirect:   model.PostLogoutRedirectURIList(def.Consoles.PlatformAdminWeb.PostLogoutRedirectURIs),
			backChannelLogoutURI: def.Consoles.PlatformAdminWeb.BackChannelLogoutURI,
		},
		{
			code:                 oauthClientTenantAdminWeb,
			name:                 "租户管理后台",
			appID:                tenantAdminApp.ID,
			redirectURIs:         model.RedirectURIList(def.Consoles.TenantAdminWeb.RedirectURIs),
			postLogoutRedirect:   model.PostLogoutRedirectURIList(def.Consoles.TenantAdminWeb.PostLogoutRedirectURIs),
			backChannelLogoutURI: def.Consoles.TenantAdminWeb.BackChannelLogoutURI,
		},
	}
	for _, cd := range clientDefs {
		// 编码规则在种子入口先行校验：内置 client_id 同时是网关侧的 audience 白名单值，
		// 写错一个字符就会让该控制台的令牌全部 401，宁可阻断启动也不要落库。
		if !model.IsValidClientCode(cd.code) {
			return fmt.Errorf("seed oauth client 编码 %q 不符合规则 %s", cd.code, model.ClientCodePattern)
		}
		entity, err := findApplicationClientByCode(db, cd.code)
		if err != nil {
			return err
		}
		if entity != nil {
			// 已存在即返回：name/回调地址/授权类型/令牌 TTL 等全部归运维（create_only 语义），
			// 引导不回写任何字段；source/app_id 是安全不变式，由控制台拒写点保证。
			continue
		}
		entity = &model.ApplicationClientEntity{
			TenantID:                tenant.ID,
			AppID:                   cd.appID,
			Code:                    cd.code,
			Name:                    cd.name,
			RedirectURIs:            cd.redirectURIs,
			PostLogoutRedirectURIs:  cd.postLogoutRedirect,
			BackChannelLogoutURI:    cd.backChannelLogoutURI,
			GrantTypes:              seedOIDCClientGrantTypes,
			ResponseTypes:           model.ResponseTypeList{model.ResponseTypeCode},
			TokenEndpointAuthMethod: model.TokenEndpointAuthMethodNone,
			RequirePKCE:             model.ClientPKCEPolicyEnable,
			DefaultScopes:           model.DefaultScopeList{model.ScopeOpenID, model.ScopeProfile, model.ScopeEmail},
			Source:                  model.ApplicationClientSourceBuiltin,
			Status:                  model.ApplicationClientStatusEnable,
		}
		if err := db.WithContext(ctx).Create(entity).Error; err != nil {
			return fmt.Errorf("seed oauth client %s create fail: %w", cd.code, err)
		}
		glog.Infof(ctx, "[seed] oauth client created, code:%s", cd.code)
		rep.created(model.SeedEntityApplicationClient, cd.code)
	}
	return nil
}
