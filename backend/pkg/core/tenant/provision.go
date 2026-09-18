package tenant

import (
	"context"
	"fmt"

	"github.com/morehao/ark-iam/pkg/core/user"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
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
	// ProvisionRoleName 内置租户管理员角色名称（幂等定位键为 (tenant_id, app_id, source=builtin)）。
	ProvisionRoleName = "租户管理员"
	// ProvisionRoleCode 内置租户管理员角色编码：跨系统授权契约值，也是 OIDC ID token `groups`
	// 的取值（下游按「前缀 + 编码」认策略名）。中性命名，不写下游产品语义。
	ProvisionRoleCode = model.RoleCodeTenantAdmin
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
	// 开通目标租户由 req.TenantID 指定，可能与调用方当前租户不同（平台侧/协议层建租户）：
	// 显式声明「指定租户」作用域，让本函数不依赖调用方 ctx 恰好携带目标租户。
	ctx = dbclient.ExplicitTenantContext(ctx, req.TenantID)

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

	// 2.1 角色模板物化：订阅建立即把该应用的角色模板落到本租户（内置应用的模板通常为空，
	// 是幂等空操作）。产品锚点角色不受影响——模板撤下逻辑显式跳过锚点编码。
	if err := SyncAppRoleTemplate(ctx, tx, &SyncAppRoleTemplateReq{
		TenantID:  req.TenantID,
		AppID:     app.ID,
		CreatedBy: req.CreatedBy,
	}); err != nil {
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
// 幂等定位键为 (tenant_id, app_id, source=builtin)；编码按 create_only 语义只在创建时写入。
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
			Code:        ProvisionRoleCode,
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

	// 后续成员/权限写入都属于"刚建出来的这个租户"：租户 ID 在本行之前才生成，
	// 因此显式声明「指定租户」作用域，既覆盖请求 ctx 无作用域（协议层建租户），
	// 也覆盖请求 ctx 携带的是平台租户（平台侧建租户）——两者都不该影响新租户的写入。
	newTenantCtx := dbclient.ExplicitTenantContext(ctx, tenantEntity.ID)

	adminReq := *req.AdminUser
	adminReq.TenantID = tenantEntity.ID
	adminReq.Source = model.UserSourceBuiltin
	adminReq.IsOwner = true
	adminReq.PrimaryDepartmentID = rootDept.ID
	adminUser, personCreated, err := user.Create(newTenantCtx, tx, &adminReq)
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

// SyncAppRoleTemplateReq 构造 SyncAppRoleTemplate 入参。
type SyncAppRoleTemplateReq struct {
	TenantID  string
	AppID     string
	CreatedBy string
}

// SyncAppRoleTemplateToTenantsReq 构造 SyncAppRoleTemplateToTenants 入参。
type SyncAppRoleTemplateToTenantsReq struct {
	AppID     string
	CreatedBy string
}

// SyncAppRoleTemplate 在 tx 事务内把某应用的**角色模板**（application.role_template）物化到指定租户。
//
// 模板是跨系统授权契约值的唯一来源（产品锚点除外），物化规则：
//
//  1. **模板新增/改名**：按 (tenant_id, app_id, code) 定位，命中 source=builtin 的行则回写名称，
//     缺失则新建（source=builtin、admin_type=normal）；命中 source=custom 的同码存量行只告警跳过
//     ——不劫持租户自建角色（新模型下租户已无法写入 code，这只防御存量数据）。
//  2. **模板移除**：该应用下 source=builtin 且 code 不在模板中的行（产品锚点除外）连同
//     user_role / role_menu 关联一并删除。契约值撤下必须是真撤下：否则被撤的 code 仍在 ID token
//     的 groups 里，继续拿到下游策略（下游按「前缀 + 编码」认策略名）。
//
// 幂等：可重复执行，不产生重复行（role 表无唯一索引，幂等由应用层定位保证）。
// 必须在调用方的事务 tx 内执行（tx 为空返回错误）；应用不存在返回错误让调用方回滚。
func SyncAppRoleTemplate(ctx context.Context, tx *gorm.DB, req *SyncAppRoleTemplateReq) error {
	if tx == nil {
		return fmt.Errorf("core/tenant: tx is required")
	}
	if req == nil || req.TenantID == "" || req.AppID == "" {
		return fmt.Errorf("core/tenant: tenant id and app id are required")
	}
	// 目标租户显式声明，不依赖调用方 ctx 恰好携带目标租户（平台侧 fan-out 时 ctx 是跨租户作用域）。
	ctx = dbclient.ExplicitTenantContext(ctx, req.TenantID)

	app, err := dao.NewApplicationDao().WithTx(tx).GetByID(ctx, req.AppID)
	if err != nil {
		return fmt.Errorf("query application %s fail: %w", req.AppID, err)
	}
	if app == nil || app.ID == "" {
		return fmt.Errorf("application %s not found", req.AppID)
	}
	template := app.RoleTemplateList()

	roleDao := dao.NewRoleDao().WithTx(tx)
	for _, item := range template {
		// 产品锚点（platform_admin/tenant_admin）属平台自身的策略命名空间，模板永远不得定义或覆盖它们。
		// 写入侧（svcapplication）已拒绝锚点编码，这里再挡一次：role_template 是普通 JSON 列，
		// 直连改库/历史数据都可能绕过写入侧校验。
		if model.IsProductAnchorRoleCode(item.Code) {
			glog.Warnf(ctx, "[tenant.SyncAppRoleTemplate] product anchor code in template, skip, tenantID:%s, appID:%s, code:%s",
				req.TenantID, req.AppID, item.Code)
			continue
		}
		existing, err := roleDao.GetByCond(ctx, &dao.RoleCond{
			TenantID: req.TenantID,
			AppID:    req.AppID,
			Code:     item.Code,
		})
		if err != nil {
			return fmt.Errorf("query role %s fail: %w", item.Code, err)
		}
		if existing == nil || existing.ID == "" {
			if err := roleDao.Insert(ctx, &model.RoleEntity{
				TenantID:    req.TenantID,
				AppID:       req.AppID,
				Name:        item.Name,
				Code:        item.Code,
				Source:      model.RoleSourceBuiltin,
				AdminType:   model.SysAdminTypeNormal,
				CreatedBy:   req.CreatedBy,
				Description: fmt.Sprintf("%s（应用角色模板下发）", app.Name),
			}); err != nil {
				return fmt.Errorf("insert template role %s fail: %w", item.Code, err)
			}
			continue
		}
		if existing.Source != model.RoleSourceBuiltin {
			glog.Warnf(ctx, "[tenant.SyncAppRoleTemplate] role code occupied by custom role, skip, tenantID:%s, appID:%s, code:%s",
				req.TenantID, req.AppID, item.Code)
			continue
		}
		if existing.Name == item.Name {
			continue
		}
		if err := roleDao.UpdateMap(ctx, existing.ID, map[string]any{
			"name":       item.Name,
			"updated_by": req.CreatedBy,
		}); err != nil {
			return fmt.Errorf("update template role %s fail: %w", item.Code, err)
		}
	}

	return withdrawStaleTemplateRoles(ctx, tx, req, template)
}

// withdrawStaleTemplateRoles 撤下应用内已从模板中移除的模板角色（产品锚点不在此列，见调用方注释）。
func withdrawStaleTemplateRoles(ctx context.Context, tx *gorm.DB, req *SyncAppRoleTemplateReq, template model.RoleTemplateItemList) error {
	stale, err := dao.NewRoleDao().WithTx(tx).GetListByCond(ctx, &dao.RoleCond{
		TenantID: req.TenantID,
		AppID:    req.AppID,
		Source:   model.RoleSourceBuiltin,
	})
	if err != nil {
		return fmt.Errorf("query builtin roles fail: %w", err)
	}
	for i := range stale {
		role := stale[i]
		if role.Code == "" || model.IsProductAnchorRoleCode(role.Code) || template.HasCode(role.Code) {
			continue
		}
		if err := deleteRoleWithRelations(ctx, tx, req.TenantID, role.ID, req.CreatedBy); err != nil {
			return fmt.Errorf("withdraw template role %s fail: %w", role.Code, err)
		}
		glog.Infof(ctx, "[tenant.SyncAppRoleTemplate] template role withdrawn, tenantID:%s, appID:%s, code:%s",
			req.TenantID, req.AppID, role.Code)
	}
	return nil
}

// deleteRoleWithRelations 删除角色并清理其 user_role / role_menu 关联（与租户侧删角色同一口径）。
func deleteRoleWithRelations(ctx context.Context, tx *gorm.DB, tenantID, roleID, operatorID string) error {
	if err := dao.NewRoleDao().WithTx(tx).Delete(ctx, roleID, operatorID); err != nil {
		return err
	}
	userRoles, err := dao.NewUserRoleDao().WithTx(tx).GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, RoleID: roleID})
	if err != nil {
		return err
	}
	for _, r := range userRoles {
		if err := dao.NewUserRoleDao().WithTx(tx).Delete(ctx, r.ID, operatorID); err != nil {
			return err
		}
	}
	roleMenus, err := dao.NewRoleMenuDao().WithTx(tx).GetListByCond(ctx, &dao.RoleMenuCond{TenantID: tenantID, RoleID: roleID})
	if err != nil {
		return err
	}
	for _, r := range roleMenus {
		if err := dao.NewRoleMenuDao().WithTx(tx).Delete(ctx, r.ID, operatorID); err != nil {
			return err
		}
	}
	return nil
}

// SyncAppRoleTemplateToTenants 在 tx 事务内把某应用的角色模板同步到**所有订阅了该应用的租户**：
// 平台侧改模板后调用一次，避免已开通租户的角色集合与新模板漂移。
// tenant_application 是租户表，跨租户列举必须显式声明跨租户作用域（见 AGENTS「上下文传递与租户作用域」）。
func SyncAppRoleTemplateToTenants(ctx context.Context, tx *gorm.DB, req *SyncAppRoleTemplateToTenantsReq) error {
	if tx == nil {
		return fmt.Errorf("core/tenant: tx is required")
	}
	if req == nil || req.AppID == "" {
		return fmt.Errorf("core/tenant: app id is required")
	}
	subs, err := dao.NewTenantApplicationDao().WithTx(tx).GetListByCond(
		dbclient.CrossTenantContext(ctx),
		&dao.TenantApplicationCond{AppID: req.AppID},
	)
	if err != nil {
		return fmt.Errorf("query tenant_application fail: %w", err)
	}
	for i := range subs {
		if err := SyncAppRoleTemplate(ctx, tx, &SyncAppRoleTemplateReq{
			TenantID:  subs[i].TenantID,
			AppID:     req.AppID,
			CreatedBy: req.CreatedBy,
		}); err != nil {
			return fmt.Errorf("sync app role template to tenant %s: %w", subs[i].TenantID, err)
		}
	}
	return nil
}
