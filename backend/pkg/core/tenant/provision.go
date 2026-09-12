package tenant

import (
	"context"
	"fmt"

	"github.com/morehao/ark-iam/pkg/core/user"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/glog"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// 租户自服务权限开通的内置定义（单一事实源）：
// pkg/seed（平台租户 bootstrap）与各建租户链路共用同一份编码，避免"角色/菜单集合"出现两份定义而漂移。
const (
	// ProvisionAppSeedKey 租户管理后台（租户自服务控制台）应用的**种子身份键**（model.ApplicationEntity.SeedKey）。
	// 开通链路按它定位应用，而不是按 application.code——code 是运营可改的业务标识；
	// 该值与种子定义（pkg/seed 的 appCodeTenantAdmin）一致，改定义时两处同步。
	ProvisionAppSeedKey = "tenant_admin"
	// ProvisionRoleName 内置租户管理员角色名称（角色无业务编码，(tenant_id, app_id, source=builtin) 即其业务唯一键）。
	ProvisionRoleName = "租户管理员"
	// ProvisionRoleDesc 内置租户管理员角色描述。
	ProvisionRoleDesc = "租户管理后台应用管理员，拥有全部租户管理后台权限"
	// ProvisionAdminType 内置管理员角色的系统管理类型（租户 tenant_admin 与平台 admin 种子共用；admin=具备系统管理能力）。
	ProvisionAdminType = model.SysAdminTypeAdmin
)

// ProvisionMenuSeedKeys 内置租户管理员默认授权的**菜单种子身份键**（tenant_admin 应用下的叶子菜单，见 model.MenuEntity.SeedKey）。
// 用身份键而非菜单编码：运营在控制台改过菜单 code 之后，新开通的租户仍能授权到同一批菜单。
var ProvisionMenuSeedKeys = []string{"department", "tenant-user", "tenant-role", "tenant-api-key"}

// ProvisionTenantAdminReq 构造 ProvisionTenantAdmin 入参。
type ProvisionTenantAdminReq struct {
	TenantID    string
	GrantUserID string // 可选：把内置角色授予该用户（建租户管理员/owner 时传入）
	CreatedBy   string
}

// ProvisionTenantAdmin 在 tx 事务内为新租户开通租户自服务权限：
// 应用订阅（tenant_application）+ 内置管理员角色（role）+ 角色菜单授权（role_menu）+
// 管理员角色绑定（user_role）。全部步骤应用层查重后 upsert，可重复执行且不产生重复行
// （这四张表均无唯一索引，故幂等只能由应用层保证）。
//
// 必须在调用方的事务 tx 内执行（tx 为空返回错误）。
// 应用/菜单属全局种子数据：应用缺失视为种子未就绪，直接返回错误让调用方回滚；
// 单个菜单缺失只告警跳过（菜单可能已下线，不应因此阻断建租户）。
func ProvisionTenantAdmin(ctx context.Context, tx *gorm.DB, req *ProvisionTenantAdminReq) (*model.RoleEntity, error) {
	if tx == nil {
		return nil, fmt.Errorf("core/tenant: tx is required")
	}
	if req == nil || req.TenantID == "" {
		return nil, fmt.Errorf("core/tenant: tenant id is required")
	}

	// 1. 定位租户管理后台应用（全局种子数据，缺失即种子未跑完）：按种子身份键定位，
	// 运营改过 application.code 后依然命中同一行。
	app, err := dao.NewApplicationDao().WithTx(tx).GetByCond(ctx, &dao.ApplicationCond{SeedKey: ProvisionAppSeedKey})
	if err != nil {
		return nil, fmt.Errorf("query application %s fail: %w", ProvisionAppSeedKey, err)
	}
	if app == nil || app.ID == "" {
		return nil, fmt.Errorf("application %s not found (seed not ready)", ProvisionAppSeedKey)
	}

	// 2. 应用订阅：新租户默认开通（已存在则保持现状，不覆盖租户侧的启停选择）
	if err := ensureTenantApplication(ctx, tx, req, app.ID); err != nil {
		return nil, err
	}

	// 3. 内置管理员角色（source=builtin、admin_type=admin）
	role, err := ensureBuiltinRole(ctx, tx, req, app.ID)
	if err != nil {
		return nil, err
	}

	// 4. 角色菜单授权
	if err := ensureRoleMenus(ctx, tx, req, role.ID); err != nil {
		return nil, err
	}

	// 5. 把角色授予指定用户（建租户管理员时）
	if req.GrantUserID != "" {
		if err := ensureUserRole(ctx, tx, req, role.ID); err != nil {
			return nil, err
		}
	}
	return role, nil
}

// ensureTenantApplication 幂等写入租户应用订阅。
func ensureTenantApplication(ctx context.Context, tx *gorm.DB, req *ProvisionTenantAdminReq, appID string) error {
	appDao := dao.NewTenantApplicationDao().WithTx(tx)
	existing, err := appDao.GetByCond(ctx, &dao.TenantApplicationCond{TenantID: req.TenantID, AppID: appID})
	if err != nil {
		return fmt.Errorf("query tenant_application fail: %w", err)
	}
	if existing != nil && existing.ID != "" {
		return nil
	}
	entity := &model.TenantApplicationEntity{
		TenantID:     req.TenantID,
		AppID:        appID,
		Status:       model.TenantApplicationStatusEnable,
		Config:       datatypes.JSON([]byte(`{}`)),
		GrantedScope: datatypes.JSON([]byte(`[]`)),
		CreatedBy:    req.CreatedBy,
	}
	if err := appDao.Insert(ctx, entity); err != nil {
		return fmt.Errorf("insert tenant_application fail: %w", err)
	}
	return nil
}

// ensureBuiltinRole 幂等写入内置租户管理员角色，并回填 admin_type（存量数据可能被改错）。
// 角色无业务编码，幂等定位键为 (tenant_id, app_id, source=builtin)。
func ensureBuiltinRole(ctx context.Context, tx *gorm.DB, req *ProvisionTenantAdminReq, appID string) (*model.RoleEntity, error) {
	roleDao := dao.NewRoleDao().WithTx(tx)
	builtinSource := model.RoleSourceBuiltin
	role, err := roleDao.GetByCond(ctx, &dao.RoleCond{
		TenantID: req.TenantID,
		AppID:    appID,
		Source:   builtinSource,
	})
	if err != nil {
		return nil, fmt.Errorf("query builtin role fail: %w", err)
	}
	adminType := ProvisionAdminType
	if role == nil || role.ID == "" {
		role = &model.RoleEntity{
			TenantID:    req.TenantID,
			AppID:       appID,
			Name:        ProvisionRoleName,
			Description: ProvisionRoleDesc,
			Source:      builtinSource,
			AdminType:   adminType,
			CreatedBy:   req.CreatedBy,
		}
		if err := roleDao.Insert(ctx, role); err != nil {
			return nil, fmt.Errorf("insert builtin role fail: %w", err)
		}
		return role, nil
	}
	// 幂等回填：确保内置管理员角色不会被误置为普通类型
	if role.AdminType != adminType {
		if err := roleDao.UpdateMap(ctx, role.ID, map[string]any{"admin_type": adminType}); err != nil {
			return nil, fmt.Errorf("update builtin role fail: %w", err)
		}
		role.AdminType = adminType
	}
	return role, nil
}

// ensureRoleMenus 幂等写入角色-菜单授权；菜单缺失只告警跳过。
// 按菜单的**种子身份键**定位（不用菜单编码，也不限定应用）：运营改过菜单 code
// 或把内置菜单移到别的应用后，新开通租户仍授权到同一批菜单。
func ensureRoleMenus(ctx context.Context, tx *gorm.DB, req *ProvisionTenantAdminReq, roleID string) error {
	menuDao := dao.NewMenuDao().WithTx(tx)
	roleMenuDao := dao.NewRoleMenuDao().WithTx(tx)
	for _, menuSeedKey := range ProvisionMenuSeedKeys {
		menu, err := menuDao.GetByCond(ctx, &dao.MenuCond{SeedKey: menuSeedKey})
		if err != nil {
			return fmt.Errorf("query menu %s fail: %w", menuSeedKey, err)
		}
		if menu == nil || menu.ID == "" {
			glog.Warnf(ctx, "[tenant.ProvisionTenantAdmin] menu not found, skip grant, tenantID:%s, seedKey:%s", req.TenantID, menuSeedKey)
			continue
		}
		existing, err := roleMenuDao.GetByCond(ctx, &dao.RoleMenuCond{TenantID: req.TenantID, RoleID: roleID, MenuID: menu.ID})
		if err != nil {
			return fmt.Errorf("query role_menu %s fail: %w", menuSeedKey, err)
		}
		if existing != nil && existing.ID != "" {
			continue
		}
		if err := roleMenuDao.Insert(ctx, &model.RoleMenuEntity{
			TenantID:  req.TenantID,
			RoleID:    roleID,
			MenuID:    menu.ID,
			CreatedBy: req.CreatedBy,
		}); err != nil {
			return fmt.Errorf("insert role_menu %s fail: %w", menuSeedKey, err)
		}
	}
	return nil
}

// ensureUserRole 幂等绑定用户与角色。
func ensureUserRole(ctx context.Context, tx *gorm.DB, req *ProvisionTenantAdminReq, roleID string) error {
	userRoleDao := dao.NewUserRoleDao().WithTx(tx)
	existing, err := userRoleDao.GetByCond(ctx, &dao.UserRoleCond{
		TenantID: req.TenantID,
		UserID:   req.GrantUserID,
		RoleID:   roleID,
	})
	if err != nil {
		return fmt.Errorf("query user_role fail: %w", err)
	}
	if existing != nil && existing.ID != "" {
		return nil
	}
	if err := userRoleDao.Insert(ctx, &model.UserRoleEntity{
		TenantID:  req.TenantID,
		UserID:    req.GrantUserID,
		RoleID:    roleID,
		CreatedBy: req.CreatedBy,
	}); err != nil {
		return fmt.Errorf("insert user_role fail: %w", err)
	}
	return nil
}

// CreateTenantWithBuiltinAdminReq 构造 CreateTenantWithBuiltinAdmin 入参。
type CreateTenantWithBuiltinAdminReq struct {
	Tenant *CreateWithRootDeptReq // 租户与根部门定义（编码/名称/类型等）
	// AdminUser 内置管理员的用户定义；其中 TenantID、PrimaryDepartmentID、Source 由本函数统一覆盖：
	// 租户必为新建租户、行政主部门必为新建根部门、来源必为 builtin（D2/D4）。
	AdminUser *user.CreateReq
}

// CreateTenantWithBuiltinAdminResult 建租户链路的产出（调用方决定如何回显/审计）。
type CreateTenantWithBuiltinAdminResult struct {
	Tenant             *model.TenantEntity
	RootDept           *model.DepartmentEntity
	AdminUser          *model.UserEntity
	AdminPersonCreated bool // 内置管理员的自然人是否为本次新建（决定是否回显初始临时密码）
}

// CreateTenantWithBuiltinAdmin 在 tx 事务内一次性完成"建租户 + 内置管理员 + 租户自服务权限开通"：
//
//	租户 + 同名根部门（CreateWithRootDept）
//	→ 内置管理员用户（user.Create，source=builtin、is_owner、归属根部门）
//	→ 权限开通（ProvisionTenantAdmin：应用订阅/内置角色/菜单授权/角色绑定）
//
// 平台建租户（管理员为新建自然人，持临时密码）与自助建租户（管理员即当前登录自然人）
// 只差 AdminUser 的定义，其余步骤完全一致，故合并为同一实现，避免两条链路各自漂移。
//
// 必须在调用方的事务 tx 内执行（tx 为空返回错误）；任一步失败即整体回滚，不产出半成品租户
// （半成品租户会出现"能登录但没有管理员/没有权限"的死局）。
func CreateTenantWithBuiltinAdmin(ctx context.Context, tx *gorm.DB, req *CreateTenantWithBuiltinAdminReq) (*CreateTenantWithBuiltinAdminResult, error) {
	if tx == nil {
		return nil, fmt.Errorf("core/tenant: tx is required")
	}
	if req == nil || req.Tenant == nil {
		return nil, fmt.Errorf("core/tenant: tenant is required")
	}
	if req.AdminUser == nil {
		return nil, fmt.Errorf("core/tenant: admin user is required")
	}

	tenantEntity, rootDept, err := CreateWithRootDept(ctx, tx, req.Tenant)
	if err != nil {
		return nil, fmt.Errorf("create tenant with root dept: %w", err)
	}

	adminReq := *req.AdminUser
	adminReq.TenantID = tenantEntity.ID
	adminReq.Source = model.UserSourceBuiltin
	adminReq.IsOwner = true
	adminReq.PrimaryDepartmentID = rootDept.ID
	adminUser, personCreated, err := user.Create(ctx, tx, &adminReq)
	if err != nil {
		return nil, fmt.Errorf("create tenant admin user: %w", err)
	}

	if _, err := ProvisionTenantAdmin(ctx, tx, &ProvisionTenantAdminReq{
		TenantID:    tenantEntity.ID,
		GrantUserID: adminUser.ID,
		CreatedBy:   adminReq.CreatedBy,
	}); err != nil {
		return nil, fmt.Errorf("provision tenant admin: %w", err)
	}

	return &CreateTenantWithBuiltinAdminResult{
		Tenant:             tenantEntity,
		RootDept:           rootDept,
		AdminUser:          adminUser,
		AdminPersonCreated: personCreated,
	}, nil
}
