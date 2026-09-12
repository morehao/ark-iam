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
	"reflect"
	"time"

	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/model"
	// 别名：SeedIam 内以 tenant 命名的局部变量会遮蔽同名包
	iamtenant "github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

const (
	// tenantCodePlatform 平台租户（平台运营中心）编码：取自字段权威矩阵所在包（pkg/model），
	// 控制台"平台租户不可挂起"的判定与本处的定位共用同一份常量。形态与自动生成规则
	// （pkg/core/tenant.GenerateCode：t_<12 位随机 hex>）一致——同前缀、后缀可读且固定。
	// 自动生成编码的随机段只用小写 hex，"platform" 含非 hex 字符，故两者永不冲突。
	tenantCodePlatform = model.SeedPlatformTenantCode
	// tenantCodePlatformLegacy 历史种子编码（旧版本为 "platform"）。
	// 启动时若命中该编码的平台租户，原地改名（保留主键，避免租户重建导致引用失联）。
	tenantCodePlatformLegacy = "platform"

	// tenantNamePlatform 平台租户名称（种子定义）：仅用于创建与一次性迁移目标值。
	// 该字段在矩阵里是 migrate_once——运维把平台租户改成自己的公司名后，种子不再回写。
	tenantNamePlatform = "平台运营中心"

	// seedAdvisoryLockKey 播种互斥键（Postgres 事务级 advisory lock）：分体部署时四个应用会同时
	// 启动播种，用它把执行串行化，避免并发插入撞关联表唯一索引后中断启动。取值仅需全系统一致。
	seedAdvisoryLockKey = int64(0x61726b5f69616d)

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

// seedMigration 一次性改名条目：仅当字段当前值等于 from 时改写为 to（值匹配，不改运维自定义值）。
type seedMigration struct {
	entity string
	field  string
	from   string
	to     string
}

// seedMigrations 历史改名清单（migrate_once 语义的唯一登记处）：
// 只登记"跨版本必须自愈的核心标识/展示名"，编码类改名（platform→t_platform、
// platform-admin→platform_admin）因改变后续查询键，仍在各自的 upsert 里先行处理。
// 同一字段可累积多条（A→B、B→C），migrateString 会在一次启动内链式应用。
var seedMigrations = []seedMigration{
	{model.SeedEntityTenant, "name", "Default Tenant", tenantNamePlatform},
	// 根部门与平台租户同名（派生），随租户名的历史改名同步一次
	{model.SeedEntityDepartment, "name", "Default Tenant", tenantNamePlatform},
}

// migrateString 对单值应用 seedMigrations 的值匹配迁移，返回新值与是否发生迁移。
// 链式条目（A→B、B→C）在一次调用内连续应用；无匹配即为空操作（幂等，可重复执行）。
func migrateString(entity, field, current string) (string, bool) {
	value, migrated := current, false
	for round := 0; round <= len(seedMigrations); round++ {
		matched := false
		for _, migration := range seedMigrations {
			if migration.entity == entity && migration.field == field && migration.from == value {
				value, matched, migrated = migration.to, true, true
				break
			}
		}
		if !matched {
			break
		}
	}
	return value, migrated
}

// reconcileFields 按字段权威矩阵收敛实体字段：只写矩阵声明为 reconcile、且当前值不一致的字段。
// 矩阵（pkg/model.SeedFieldAuthorities）是唯一真相源——新增收敛字段必须先声明，否则不会被写入。
// 返回字段级 from→to 变更（供 Report 与启动日志），无变更时返回 nil。
func reconcileFields(ctx context.Context, db *gorm.DB, entity, table, id string, current, desired map[string]any) (map[string]string, error) {
	updateMap := make(map[string]any)
	changes := make(map[string]string)
	for field, want := range desired {
		if !model.SeedOwnsField(entity, field) {
			continue
		}
		if reflect.DeepEqual(current[field], want) {
			continue
		}
		updateMap[field] = want
		changes[field] = fmt.Sprintf("%v -> %v", current[field], want)
	}
	if len(updateMap) == 0 {
		return nil, nil
	}
	if err := db.WithContext(ctx).Table(table).Where("id = ?", id).Updates(updateMap).Error; err != nil {
		return nil, err
	}
	return changes, nil
}

// Change 一次种子变更：created（新建）/ updated（收敛）/ migrated（一次性改名）。
type Change struct {
	Entity string
	Key    string
	Action string
	Fields map[string]string // 字段级 from→to（created 时为空）
}

// Report 本次种子执行的变更报告：仅用于启动日志核对（不落库），
// 覆盖权威矩阵相关实体与平台自举产物（租户/部门/应用/菜单/客户端/角色/订阅/管理员）。
type Report struct {
	Changes []Change
}

func (r *Report) created(entity, key string) {
	r.Changes = append(r.Changes, Change{Entity: entity, Key: key, Action: "created"})
}

func (r *Report) updated(entity, key string, fields map[string]string) {
	if len(fields) == 0 {
		return
	}
	r.Changes = append(r.Changes, Change{Entity: entity, Key: key, Action: "updated", Fields: fields})
}

func (r *Report) migrated(entity, key, field, from, to string) {
	r.Changes = append(r.Changes, Change{
		Entity: entity, Key: key, Action: "migrated",
		Fields: map[string]string{field: from + " -> " + to},
	})
}

// Summary 汇总各动作计数（启动日志一行）。
func (r *Report) Summary() string {
	counts := map[string]int{}
	for _, change := range r.Changes {
		counts[change.Action]++
	}
	return fmt.Sprintf("created=%d updated=%d migrated=%d", counts["created"], counts["updated"], counts["migrated"])
}

// log 输出变更报告：汇总一行；改名条目单独 Warn 级输出，便于部署时核对"这次启动改了什么"。
func (r *Report) log(ctx context.Context) {
	if len(r.Changes) == 0 {
		return
	}
	glog.Infof(ctx, "[seed] done, %s", r.Summary())
	for _, change := range r.Changes {
		if change.Action != "migrated" {
			continue
		}
		glog.Warnf(ctx, "[seed] migrated %s(%s) fields:%v", change.Entity, change.Key, change.Fields)
	}
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

// SeedIam 幂等写入 IAM 基础种子数据。任一环节失败即返回错误，由调用方决定是否阻断启动。
func SeedIam(ctx context.Context, db *gorm.DB) error {
	_, err := Run(ctx, db)
	return err
}

// Run 在单事务内执行种子并返回本次变更报告：
//   - 整体事务：任一环节失败即回滚，不留"半播"状态；重入时所有分支都以当前值为条件，天然幂等；
//   - 进程间互斥：Postgres 取 advisory lock，串行化多进程并发播种（SQLite 等测试库跳过）。
func Run(ctx context.Context, db *gorm.DB) (Report, error) {
	rep := Report{}
	txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSeed(tx); err != nil {
			return err
		}
		return seedAll(ctx, tx, &rep)
	})
	if txErr != nil {
		return rep, txErr
	}
	rep.log(ctx)
	return rep, nil
}

// seedAll 顺序执行各类种子（在 Run 的单个事务内）。步骤编号与依赖顺序一一对应。
func seedAll(ctx context.Context, db *gorm.DB, rep *Report) error {
	// 1. 平台租户
	tenant, err := getOrCreateTenant(ctx, db, rep)
	if err != nil {
		return err
	}

	// 2. 租户同名顶级部门（用户归属的根部门，管理员也归属于此）
	rootDept, err := seedRootDepartment(ctx, db, tenant, rep)
	if err != nil {
		return err
	}

	// 3. 应用（历史库的连字符编码由 getOrCreateApplication 原地改名）
	adminApp, err := getOrCreateApplication(ctx, db, rep, appCodeAdmin, appCodeAdminLegacy, "平台管理后台", "平台管理后台应用", 0, model.AppSourceBuiltin)
	if err != nil {
		return err
	}
	tenantAdminApp, err := getOrCreateApplication(ctx, db, rep, appCodeTenantAdmin, appCodeTenantAdminLegacy, "租户管理后台", "租户管理后台应用", 1, model.AppSourceBuiltin)
	if err != nil {
		return err
	}

	// 4. 角色（admin 归属平台管理后台；tenant_admin 由第 10 步的权限开通统一创建）
	adminRole, err := seedRoles(ctx, db, rep, tenant, adminApp)
	if err != nil {
		return err
	}

	// 5. 菜单
	menus, err := seedMenus(ctx, db, rep, adminApp, tenantAdminApp)
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

	// 7. 角色-菜单关联（仅平台管理后台 admin 角色；tenant_admin 由第 10 步开通时授权）
	if err := seedRoleMenus(ctx, db, tenant, adminRole, menus); err != nil {
		return err
	}

	// 8. 租户应用订阅（平台管理后台 platform_admin；租户管理后台 tenant_admin 由第 10 步开通时订阅）
	if err := seedTenantApplications(ctx, db, rep, tenant, adminApp); err != nil {
		return err
	}

	// 9. 默认管理员（person + user + 顶级部门归属）
	adminUser, err := seedAdminUser(ctx, db, rep, tenant, rootDept)
	if err != nil {
		return err
	}
	if err := seedAdminUserRole(ctx, db, rep, tenant, adminUser, adminRole); err != nil {
		return err
	}

	// 10. 平台租户的租户自服务权限开通：与"新建租户"共用同一实现
	// （pkg/core/tenant.ProvisionTenantAdmin），保证内置角色/菜单授权/订阅只有一份定义。
	if _, err := iamtenant.ProvisionTenantAdmin(ctx, db, &iamtenant.ProvisionTenantAdminReq{
		TenantID:    tenant.ID,
		GrantUserID: adminUser.ID,
	}); err != nil {
		return fmt.Errorf("seed provision tenant admin fail: %w", err)
	}

	// 11. OIDC 测试客户端（平台管理后台客户端挂 platform_admin，租户管理后台客户端挂 tenant_admin）
	if err := seedOIDCClients(ctx, db, rep, tenant, adminApp, tenantAdminApp); err != nil {
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

func getOrCreateTenant(ctx context.Context, db *gorm.DB, rep *Report) (*model.TenantEntity, error) {
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
		// 存量库自愈（字段权威矩阵见 pkg/model/seed_authority.go）：
		//   - name 是 migrate_once：只在值仍是历史种子名（Default Tenant）时改名；
		//     运维把平台租户改成自己公司名后，种子不再回写（这是"改名"而不是"收敛"）；
		//   - status 是 reconcile：平台租户被挂起会导致整栈控制台失联，恒为 active；
		//   - type/tag/db_user 是 create_only，种子不回填。
		if name, migrated := migrateString(model.SeedEntityTenant, "name", entity.Name); migrated {
			if uErr := db.Model(&model.TenantEntity{}).Where("id = ?", entity.ID).
				Update("name", name).Error; uErr != nil {
				return nil, fmt.Errorf("seed tenant name migrate fail: %w", uErr)
			}
			glog.Infof(ctx, "[seed] tenant name migrated (%s -> %s), id:%s", entity.Name, name, entity.ID)
			rep.migrated(model.SeedEntityTenant, entity.Code, "name", entity.Name, name)
			entity.Name = name
		}
		changes, uErr := reconcileFields(ctx, db, model.SeedEntityTenant, model.TableNameTenant, entity.ID,
			map[string]any{"status": entity.Status},
			map[string]any{"status": model.TenantStatusActive})
		if uErr != nil {
			return nil, fmt.Errorf("seed tenant reconcile fail: %w", uErr)
		}
		if len(changes) > 0 {
			glog.Infof(ctx, "[seed] tenant reconciled, id:%s fields:%v", entity.ID, changes)
			rep.updated(model.SeedEntityTenant, entity.Code, changes)
			entity.Status = model.TenantStatusActive
		}
		return entity, nil
	}
	entity = &model.TenantEntity{
		Code:   tenantCodePlatform,
		Name:   tenantNamePlatform,
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
// 根部门名与租户同名是派生关系：只在值仍是历史种子名时随迁移同步一次（migrate_once），
// 运维按自己组织架构改过部门名后不再被拉回。所有种子用户（含管理员）均从属于此顶级部门。
func seedRootDepartment(ctx context.Context, db *gorm.DB, tenant *model.TenantEntity, rep *Report) (*model.DepartmentEntity, error) {
	dept := &model.DepartmentEntity{}
	err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(dept).Error
	if err == nil {
		if name, migrated := migrateString(model.SeedEntityDepartment, "name", dept.Name); migrated {
			if uErr := db.WithContext(ctx).Model(&model.DepartmentEntity{}).Where("id = ?", dept.ID).
				Update("name", name).Error; uErr != nil {
				return nil, fmt.Errorf("seed root department rename fail: %w", uErr)
			}
			glog.Infof(ctx, "[seed] root department name migrated (%s -> %s), id:%s", dept.Name, name, dept.ID)
			rep.migrated(model.SeedEntityDepartment, dept.ID, "name", dept.Name, name)
			dept.Name = name
		}
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
func getOrCreateApplication(ctx context.Context, db *gorm.DB, rep *Report, code, legacyCode, name, desc string, sort int, source model.AppSource) (*model.ApplicationEntity, error) {
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
		// 幂等回填（按字段权威矩阵）：source（内置性）+ name/description（控制台身份）归种子，
		// 每次启动收敛；控制台对这些字段已拒写（svcapplication.Update），故不存在双写者。
		// sort/status/logo_url/homepage_url 是 create_only（归运维），种子不覆盖。
		changes, uErr := reconcileFields(ctx, db, model.SeedEntityApplication, model.TableNameApplication, entity.ID,
			map[string]any{"source": entity.Source, "name": entity.Name, "description": entity.Description},
			map[string]any{"source": source, "name": name, "description": desc})
		if uErr != nil {
			return nil, fmt.Errorf("seed application %s reconcile fail: %w", code, uErr)
		}
		if len(changes) > 0 {
			glog.Infof(ctx, "[seed] application reconciled, code:%s fields:%v", code, changes)
			rep.updated(model.SeedEntityApplication, code, changes)
			entity.Source = source
			entity.Name = name
			entity.Description = desc
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
	rep.created(model.SeedEntityApplication, code)
	return entity, nil
}

// seedRoles 只种平台管理后台 admin 角色：租户管理后台的 tenant_admin 由权限开通统一创建
// （pkg/core/tenant.ProvisionTenantAdmin），建租户与种子共用同一份角色/授权定义。
// 角色无业务编码，内置角色以 (tenant_id, app_id, source=builtin) 为幂等定位键。
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

func seedMenus(ctx context.Context, db *gorm.DB, rep *Report, adminApp, tenantAdminApp *model.ApplicationEntity) (map[string]*model.MenuEntity, error) {
	defs := []seedMenu{
		// 平台管理后台：目录分组（type=directory，无页面）+ 页面叶子（type=menu，指向真实前端页面）。
		// 一级菜单按「对象域」划分（对象名词 + 中心/叶子），不使用「X 与 Y」并列命名：
		// 租户中心（租户及其资源）/ 应用中心（应用及其接入凭证）/
		// 平台管理（平台自身治理：菜单字典与审计日志）。
		// 用户与角色不再有平台端入口：两者按租户归属，读写与成员管理全部收敛到租户管理后台。
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
		// 租户管理后台一级菜单：控制台定位为「租户管理层专用」（部门/用户/角色/密钥均属管理操作，
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
			rep.created(model.SeedEntityMenu, def.code)
		} else {
			// 幂等回填（按字段权威矩阵）：内置应用菜单的结构与展示字段归种子，启动即收敛；
			// status 是 create_only（运维可停用菜单），种子不覆盖。
			changes, uErr := reconcileFields(ctx, db, model.SeedEntityMenu, model.TableNameMenu, entity.ID,
				map[string]any{
					"name":       entity.Name,
					"path":       entity.Path,
					"icon":       entity.Icon,
					"sort":       entity.Sort,
					"component":  entity.Component,
					"visibility": entity.Visibility,
					"type":       entity.Type,
					"parent_id":  entity.ParentID,
				},
				map[string]any{
					"name":       def.name,
					"path":       def.path,
					"icon":       def.icon,
					"sort":       def.sort,
					"component":  def.component,
					"visibility": visibility,
					"type":       menuTypeOf(def),
					"parent_id":  parentID,
				})
			if uErr != nil {
				return nil, fmt.Errorf("seed menu %s reconcile fail: %w", def.code, uErr)
			}
			if len(changes) > 0 {
				glog.Infof(ctx, "[seed] menu reconciled, code:%s fields:%v", def.code, changes)
				rep.updated(model.SeedEntityMenu, def.code, changes)
			}
		}
		out[def.code] = entity
	}
	return out, nil
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
		ta := &model.TenantApplicationEntity{TenantID: tenant.ID, AppID: app.ID, Status: model.TenantApplicationStatusEnable, Config: []byte(`{}`), GrantedScope: []byte(`[]`)}
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
func seedAdminUser(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, rootDept *model.DepartmentEntity) (*model.UserEntity, error) {
	if rootDept == nil || rootDept.ID == "" {
		return nil, fmt.Errorf("seed admin user fail: root department not found")
	}
	passwordHash, err := gcrypto.GeneratePasswordHash(credential.BootstrapAdminPassword)
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
		rep.created(model.SeedEntityPerson, "admin")
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
		glog.Infof(ctx, "[seed] admin user created, id:%s (default password: %s, must_change_password: false)", user.ID, credential.BootstrapAdminPassword)
		rep.created(model.SeedEntityUser, user.ID)
	}

	// 来源回填（矩阵 reconcile）：种子管理员是内置管理员，source 必须为 builtin，
	// 平台侧"重置内置管理员密码"依赖该标记定位目标用户。
	userChanges, uErr := reconcileFields(ctx, db, model.SeedEntityUser, model.TableNameUser, user.ID,
		map[string]any{"source": user.Source},
		map[string]any{"source": model.UserSourceBuiltin})
	if uErr != nil {
		return nil, fmt.Errorf("seed admin user reconcile fail: %w", uErr)
	}
	if len(userChanges) > 0 {
		glog.Infof(ctx, "[seed] admin user reconciled, id:%s fields:%v", user.ID, userChanges)
		rep.updated(model.SeedEntityUser, user.ID, userChanges)
		user.Source = model.UserSourceBuiltin
	}

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

// seedOIDCClientGrantTypes 种子 OAuth 客户端授权类型：由 model.GrantType 常量序列化，
// 避免在种子数据里裸写 JSON 字面量导致取值漂移。
var seedOIDCClientGrantTypes = func() []byte {
	b, _ := json.Marshal([]model.GrantType{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken})
	return b
}()

// seedOIDCClients 播种内置 OAuth 客户端（OIDC RP）。
// 归属应用按控制台一一对应：平台管理后台客户端 → platform_admin 应用，租户管理后台客户端 → tenant_admin 应用。
// app_id 是种子收敛字段（矩阵声明 reconcile）：存量库把两者都挂到 platform_admin 的错误绑定由此自愈。
func seedOIDCClients(ctx context.Context, db *gorm.DB, rep *Report, tenant *model.TenantEntity, adminApp, tenantAdminApp *model.ApplicationEntity) error {
	type clientDef struct {
		code                 string
		name                 string
		appID                string
		redirectURIs         string
		postLogoutRedirect   string
		backChannelLogoutURI string
	}
	defs := []clientDef{
		{
			code:                 oauthClientPlatformAdminWeb,
			name:                 "平台管理后台",
			appID:                adminApp.ID,
			redirectURIs:         `["http://localhost:4001/auth/callback"]`,
			postLogoutRedirect:   `["http://localhost:4001/login"]`,
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/platform",
		},
		{
			code:                 oauthClientTenantAdminWeb,
			name:                 "租户管理后台",
			appID:                tenantAdminApp.ID,
			redirectURIs:         `["http://localhost:4002/auth/callback"]`,
			postLogoutRedirect:   `["http://localhost:4002/login"]`,
			backChannelLogoutURI: "http://localhost:8100/oidc/bc-logout/tenant",
		},
	}
	for _, def := range defs {
		entity := &model.ApplicationClientEntity{}
		err := db.Where("code = ?", def.code).First(entity).Error
		if err == nil {
			// 幂等回填（按字段权威矩阵）：source（内置性）/ name（控制台展示名）/ app_id（归属应用，产品结构）归种子；
			// 回调地址/授权类型/令牌 TTL 等运行参数是 create_only（归运维），种子不覆盖。
			changes, uErr := reconcileFields(ctx, db, model.SeedEntityApplicationClient, model.TableNameApplicationClient, entity.ID,
				map[string]any{"source": entity.Source, "name": entity.Name, "app_id": entity.AppID},
				map[string]any{"source": model.ApplicationClientSourceBuiltin, "name": def.name, "app_id": def.appID})
			if uErr != nil {
				return fmt.Errorf("seed oauth client %s reconcile fail: %w", def.code, uErr)
			}
			if len(changes) > 0 {
				glog.Infof(ctx, "[seed] oauth client reconciled, code:%s fields:%v", def.code, changes)
				rep.updated(model.SeedEntityApplicationClient, def.code, changes)
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("seed oauth client %s query fail: %w", def.code, err)
		}
		entity = &model.ApplicationClientEntity{
			TenantID:                tenant.ID,
			AppID:                   def.appID,
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
		rep.created(model.SeedEntityApplicationClient, def.code)
	}
	return nil
}
