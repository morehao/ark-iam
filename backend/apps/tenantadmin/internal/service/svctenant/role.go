package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

func roleVisibleToTenant(entity *model.RoleEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

type RoleSvc interface {
	Create(ctx *gin.Context, req *dtotenant.RoleCreateReq) (*dtotenant.RoleCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.RoleDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.RoleUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.RoleDetailReq) (*dtotenant.RoleDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.RolePageListReq) (*dtotenant.RolePageListResp, error)
	GetMenus(ctx *gin.Context, req *dtotenant.RoleDetailReq) (*dtotenant.RoleMenuTreeResp, error)
	UpdateMenus(ctx *gin.Context, req *dtotenant.RoleMenusUpdateReq) error
}

type roleSvc struct{}

var _ RoleSvc = (*roleSvc)(nil)

func NewRoleSvc() RoleSvc {
	return &roleSvc{}
}

// Create 创建租户角色（名称租户 + 应用内唯一）。
func (svc *roleSvc) Create(ctx *gin.Context, req *dtotenant.RoleCreateReq) (*dtotenant.RoleCreateResp, error) {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.RoleCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)

	// 角色从属于租户订阅的应用：校验 appID（订阅且启用的应用均可选，含系统内置应用）
	appList, err := loadSubscribedApps(ctx)
	if err != nil {
		return nil, err
	}
	appValid := false
	for _, app := range appList {
		if app.ID == req.AppID {
			appValid = true
			break
		}
	}
	if !appValid {
		return nil, code.GetError(code.RoleCreateError)
	}

	// 角色编码是跨系统授权契约值（OIDC ID token `groups` 取值，下游按编码认策略名）：
	// 形状在 service 入口白名单校验；编码不合法/保留/重复属可预期业务边界，分别返回专用错误码。
	if !model.IsValidRoleCode(req.Code) {
		return nil, code.GetError(code.RoleCodeInvalidError)
	}
	// 系统保留编码（内置角色的下游策略锚点）不得被自建角色占用：下游按「前缀 + 编码」认策略名，
	// 前缀隔离不了同码——放开即等于允许任何租户自造一个名字撞上内置策略的角色（跨租户提权）。
	if model.IsReservedRoleCode(req.Code) {
		return nil, code.GetError(code.RoleCodeReservedError)
	}
	// 编码租户内唯一：命名空间是租户而非应用——下游只看到编码，重码会让授权含义产生歧义
	codeExists, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, Code: req.Code})
	if err != nil {
		glog.Errorf(ctx, "[svcrole.Create] query role by code fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleCreateError)
	}
	if len(codeExists) > 0 {
		return nil, code.GetError(code.RoleCodeExistsError)
	}

	// 名称应用内唯一
	existing, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, AppID: req.AppID, Name: req.Name})
	if err != nil {
		glog.Errorf(ctx, "[svcrole.Create] query role by name fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleCreateError)
	}
	if len(existing) > 0 {
		return nil, code.GetError(code.RoleCreateError)
	}

	insertEntity := &model.RoleEntity{
		TenantID:    tenantID,
		AppID:       req.AppID,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Source:      model.RoleSourceCustom,
		AdminType:   model.SysAdminTypeNormal,
		CreatedBy:   gincontext.GetUserIDString(ctx),
	}
	if err := dao.NewRoleDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcrole.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleCreateError)
	}
	return &dtotenant.RoleCreateResp{RoleID: insertEntity.ID}, nil
}

// Delete 删除角色：级联清理 user_role / role_menu 关联（事务）。
func (svc *roleSvc) Delete(ctx *gin.Context, req *dtotenant.RoleDeleteReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.RoleDeleteError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcrole.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleDeleteError)
	}
	if !roleVisibleToTenant(roleEntity, tenantID) {
		return code.GetError(code.RoleNotExistError)
	}
	// 内置角色禁止删除（防止系统管理能力失控且 seed 幂等不自动重建）
	if roleEntity.Source == model.RoleSourceBuiltin {
		return code.GetError(code.RoleDeleteBuiltinForbiddenError)
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dao.NewRoleDao().WithTx(tx).Delete(ctx, req.RoleID, userID); err != nil {
			return err
		}
		// 清理角色-用户关联
		urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, RoleID: req.RoleID})
		if err != nil {
			return err
		}
		for _, r := range urList {
			if err := dao.NewUserRoleDao().WithTx(tx).Delete(ctx, r.ID, userID); err != nil {
				return err
			}
		}
		// 清理角色-菜单关联
		rmList, err := dao.NewRoleMenuDao().GetListByCond(ctx, &dao.RoleMenuCond{TenantID: tenantID, RoleID: req.RoleID})
		if err != nil {
			return err
		}
		for _, r := range rmList {
			if err := dao.NewRoleMenuDao().WithTx(tx).Delete(ctx, r.ID, userID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcrole.Delete] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.RoleDeleteError)
	}
	return nil
}

// Update 全量更新角色。
func (svc *roleSvc) Update(ctx *gin.Context, req *dtotenant.RoleUpdateReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.RoleUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcrole.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	if !roleVisibleToTenant(roleEntity, tenantID) {
		return code.GetError(code.RoleNotExistError)
	}
	// 内置角色整体只读：它是随租户创建播种的授权锚点（编码即下游策略名取值，名称/描述由种子与运维负责），
	// 放开编辑等于允许租户自行改写下游授权语义。菜单授权不经此接口（见 UpdateMenus），不受影响。
	if roleEntity.Source == model.RoleSourceBuiltin {
		return code.GetError(code.RoleUpdateBuiltinForbiddenError)
	}

	// 自建角色编码可改（它不是本系统的定位键，改错的代价只是下游按新编码找策略，可在本控制台改回），
	// 但改动即改变下游授权：形状、保留字与租户内唯一仍必须守住，唯一性校验排除自身。
	if !model.IsValidRoleCode(req.Code) {
		return code.GetError(code.RoleCodeInvalidError)
	}
	// 保留编码对自建角色同样封闭：不得把已有自建角色改名为内置策略锚点（提权路径与创建一致）。
	// 仅在**改码时**拦截：本守卫上线前已占用保留码的存量行，仍需允许改名称/描述等其他字段。
	if req.Code != roleEntity.Code && model.IsReservedRoleCode(req.Code) {
		return code.GetError(code.RoleCodeReservedError)
	}
	codeExists, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, Code: req.Code})
	if err != nil {
		glog.Errorf(ctx, "[svcrole.Update] query role by code fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	for _, existRole := range codeExists {
		if existRole.ID != req.RoleID {
			return code.GetError(code.RoleCodeExistsError)
		}
	}

	updateMap := map[string]any{
		"code":        req.Code,
		"name":        req.Name,
		"description": req.Description,
		"updated_by":  gincontext.GetUserIDString(ctx),
	}
	if err := dao.NewRoleDao().UpdateMap(ctx, req.RoleID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcrole.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	return nil
}

// Detail 角色详情（含成员数 / 授权菜单数）。
func (svc *roleSvc) Detail(ctx *gin.Context, req *dtotenant.RoleDetailReq) (*dtotenant.RoleDetailResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcrole.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if !roleVisibleToTenant(roleEntity, tenantID) {
		return nil, code.GetError(code.RoleNotExistError)
	}
	memberCount, menuCount, err := svc.roleRelationCounts(ctx, tenantID, []string{req.RoleID})
	if err != nil {
		return nil, code.GetError(code.RoleGetDetailError)
	}
	appNameMap, err := tenantAppNameMap(ctx)
	if err != nil {
		return nil, code.GetError(code.RoleGetDetailError)
	}
	return &dtotenant.RoleDetailResp{
		RoleID:      roleEntity.ID,
		AppID:       roleEntity.AppID,
		AppName:     appNameMap[roleEntity.AppID],
		Code:        roleEntity.Code,
		Name:        roleEntity.Name,
		Description: roleEntity.Description,
		Source:      roleEntity.Source,
		AdminType:   roleEntity.AdminType,
		MemberCount: memberCount[req.RoleID],
		MenuCount:   menuCount[req.RoleID],
		CreatedAt:   roleEntity.CreatedAt.Unix(),
		UpdatedAt:   roleEntity.UpdatedAt.Unix(),
	}, nil
}

// PageList 角色分页列表（含成员数 / 授权菜单数聚合，可按应用过滤）。
func (svc *roleSvc) PageList(ctx *gin.Context, req *dtotenant.RolePageListReq) (*dtotenant.RolePageListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	cond := &dao.RoleCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:   tenantID,
		AppID:      req.AppID,
		Keyword:    req.Keyword,
		Unassigned: req.Unassigned,
	}
	roleEntityList, total, err := dao.NewRoleDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcrole.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetPageListError)
	}

	roleIDs := make([]string, 0, len(roleEntityList))
	for _, v := range roleEntityList {
		roleIDs = append(roleIDs, v.ID)
	}
	memberCount, menuCount, err := svc.roleRelationCounts(ctx, tenantID, roleIDs)
	if err != nil {
		return nil, code.GetError(code.RoleGetPageListError)
	}
	appNameMap, err := tenantAppNameMap(ctx)
	if err != nil {
		return nil, code.GetError(code.RoleGetPageListError)
	}

	list := make([]dtotenant.RolePageListItem, 0, len(roleEntityList))
	for _, v := range roleEntityList {
		list = append(list, dtotenant.RolePageListItem{
			RoleID:      v.ID,
			AppID:       v.AppID,
			AppName:     appNameMap[v.AppID],
			Code:        v.Code,
			Name:        v.Name,
			Description: v.Description,
			Source:      v.Source,
			AdminType:   v.AdminType,
			MemberCount: memberCount[v.ID],
			MenuCount:   menuCount[v.ID],
			CreatedAt:   v.CreatedAt.Unix(),
			UpdatedAt:   v.UpdatedAt.Unix(),
		})
	}
	return &dtotenant.RolePageListResp{List: list, Total: total}, nil
}

// GetMenus 角色菜单授权回显：角色所属应用的菜单树 + 已授权菜单ID（无应用归属的种子角色回退全控制台菜单）。
func (svc *roleSvc) GetMenus(ctx *gin.Context, req *dtotenant.RoleDetailReq) (*dtotenant.RoleMenuTreeResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcrole.GetMenus] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if !roleVisibleToTenant(roleEntity, tenantID) {
		return nil, code.GetError(code.RoleNotExistError)
	}

	tree, err := svc.roleMenuTree(ctx, roleEntity)
	if err != nil {
		return nil, err
	}

	rmList, err := dao.NewRoleMenuDao().GetListByCond(ctx, &dao.RoleMenuCond{TenantID: tenantID, RoleID: req.RoleID})
	if err != nil {
		glog.Errorf(ctx, "[svcrole.GetMenus] query role_menu fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleMenuGetPageListError)
	}
	menuIDs := make([]string, 0, len(rmList))
	for _, r := range rmList {
		menuIDs = append(menuIDs, r.MenuID)
	}
	return &dtotenant.RoleMenuTreeResp{List: tree, MenuIDs: menuIDs}, nil
}

// roleMenuFullTree 角色所属应用（或全控制台）的完整菜单树（含 visibility=admin，不剔除）。
func (svc *roleSvc) roleMenuFullTree(ctx *gin.Context, roleEntity *model.RoleEntity) ([]dtotenant.MenuTreeItem, error) {
	if roleEntity.AppID != "" {
		return buildAppMenuTree(ctx, roleEntity.AppID)
	}
	return buildTenantMenuTree(ctx)
}

// roleMenuTree 角色可授权的菜单树。
// 有应用归属则取该应用菜单，否则回退全租户控制台菜单（种子角色）。
// 非内置管理员角色剔除 visibility=admin 的菜单（授权约束），内置管理员角色可见全部。
func (svc *roleSvc) roleMenuTree(ctx *gin.Context, roleEntity *model.RoleEntity) ([]dtotenant.MenuTreeItem, error) {
	var tree []dtotenant.MenuTreeItem
	var err error
	if roleEntity.AppID != "" {
		tree, err = buildAppMenuTree(ctx, roleEntity.AppID)
	} else {
		tree, err = buildTenantMenuTree(ctx)
	}
	if err != nil {
		return nil, err
	}
	if roleEntity.IsBuiltinAdmin() {
		return tree, nil
	}
	return stripAdminVisibilityMenus(tree), nil
}

// stripAdminVisibilityMenus 剔除 visibility=admin 的菜单节点（父壳保留，子移除），实现普通角色授权约束。
func stripAdminVisibilityMenus(items []dtotenant.MenuTreeItem) []dtotenant.MenuTreeItem {
	result := make([]dtotenant.MenuTreeItem, 0, len(items))
	for _, item := range items {
		children := stripAdminVisibilityMenus(item.Children)
		item.Children = children
		// visibility=admin：该节点（含其 admin 子项）从本角色可选集合剔除；父壳保留以维持层级
		if model.MenuVisibility(item.Visibility) == model.MenuVisibilityAdmin {
			// 若父为 admin 但存在非 admin 子项，保留子项；否则整棵剔除
			if len(children) > 0 {
				result = append(result, item)
			}
			continue
		}
		result = append(result, item)
	}
	return result
}

// containAdminVisibilityMenus 判断菜单集合中是否含 visibility=admin 的节点。
func containAdminVisibilityMenus(tree []dtotenant.MenuTreeItem) bool {
	found := false
	var walk func(items []dtotenant.MenuTreeItem)
	walk = func(items []dtotenant.MenuTreeItem) {
		for _, m := range items {
			if model.MenuVisibility(m.Visibility) == model.MenuVisibilityAdmin {
				found = true
				return
			}
			if len(m.Children) > 0 {
				walk(m.Children)
			}
		}
	}
	walk(tree)
	return found
}

// UpdateMenus 全量替换角色菜单授权（PUT 集合语义）。
func (svc *roleSvc) UpdateMenus(ctx *gin.Context, req *dtotenant.RoleMenusUpdateReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.RoleUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcrole.UpdateMenus] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	if !roleVisibleToTenant(roleEntity, tenantID) {
		return code.GetError(code.RoleNotExistError)
	}

	// 校验菜单均属于角色所属应用（无应用归属的种子角色校验全租户控制台菜单）
	// 授权约束：非内置管理员角色不得提交 visibility=admin 菜单
	if len(req.MenuIDs) > 0 {
		fullTree, err := svc.roleMenuFullTree(ctx, roleEntity)
		if err != nil {
			return err
		}
		allowed := collectMenuIDs(fullTree)
		adminIDs := collectMenuIDs(stripAdminVisibilityMenus(fullTree)) // 非 admin 可见性菜单
		if !roleEntity.IsBuiltinAdmin() {
			for _, menuID := range req.MenuIDs {
				if allowed[menuID] && !adminIDs[menuID] {
					return code.GetError(code.RoleMenuAdminVisibilityForbiddenError)
				}
			}
		}
		for _, menuID := range req.MenuIDs {
			if !allowed[menuID] {
				return code.GetError(code.RoleMenuNotExistError)
			}
		}
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除旧关联
		oldList, err := dao.NewRoleMenuDao().GetListByCond(ctx, &dao.RoleMenuCond{TenantID: tenantID, RoleID: req.RoleID})
		if err != nil {
			return err
		}
		for _, r := range oldList {
			if err := dao.NewRoleMenuDao().WithTx(tx).Delete(ctx, r.ID, userID); err != nil {
				return err
			}
		}
		// 插入新关联
		for _, menuID := range req.MenuIDs {
			entity := &model.RoleMenuEntity{
				TenantID:  tenantID,
				RoleID:    req.RoleID,
				MenuID:    menuID,
				CreatedBy: userID,
			}
			if err := dao.NewRoleMenuDao().WithTx(tx).Insert(ctx, entity); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcrole.UpdateMenus] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.RoleMenuCreateError)
	}
	return nil
}

// roleRelationCounts 批量统计角色成员数与授权菜单数（GROUP BY，避免 N+1）。
func (svc *roleSvc) roleRelationCounts(ctx *gin.Context, tenantID string, roleIDs []string) (map[string]int64, map[string]int64, error) {
	memberCount := make(map[string]int64, len(roleIDs))
	menuCount := make(map[string]int64, len(roleIDs))
	for _, id := range roleIDs {
		memberCount[id] = 0
		menuCount[id] = 0
	}
	if len(roleIDs) == 0 {
		return memberCount, menuCount, nil
	}

	type countRow struct {
		RoleID string
		Cnt    int64
	}

	var memberRows []countRow
	if err := dbclient.IamDB(ctx).Model(&model.UserRoleEntity{}).
		Where("tenant_id = ? AND role_id IN ?", tenantID, roleIDs).
		Select("role_id, count(*) as cnt").Group("role_id").Scan(&memberRows).Error; err != nil {
		glog.Errorf(ctx, "[svcrole.roleRelationCounts] count user_role fail, err:%v", err)
		return nil, nil, err
	}
	for _, r := range memberRows {
		memberCount[r.RoleID] = r.Cnt
	}

	var menuRows []countRow
	if err := dbclient.IamDB(ctx).Model(&model.RoleMenuEntity{}).
		Where("tenant_id = ? AND role_id IN ?", tenantID, roleIDs).
		Select("role_id, count(*) as cnt").Group("role_id").Scan(&menuRows).Error; err != nil {
		glog.Errorf(ctx, "[svcrole.roleRelationCounts] count role_menu fail, err:%v", err)
		return nil, nil, err
	}
	for _, r := range menuRows {
		menuCount[r.RoleID] = r.Cnt
	}
	return memberCount, menuCount, nil
}

// collectMenuIDs 平铺菜单树收集全部菜单ID。
func collectMenuIDs(tree []dtotenant.MenuTreeItem) map[string]bool {
	result := make(map[string]bool)
	var walk func(items []dtotenant.MenuTreeItem)
	walk = func(items []dtotenant.MenuTreeItem) {
		for _, m := range items {
			result[m.MenuID] = true
			if len(m.Children) > 0 {
				walk(m.Children)
			}
		}
	}
	walk(tree)
	return result
}

// tenantAppNameMap 租户订阅应用 ID -> 名称 映射（含系统内置应用，与角色可选应用集合口径一致）。
func tenantAppNameMap(ctx *gin.Context) (map[string]string, error) {
	appList, err := loadSubscribedApps(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(appList))
	for _, app := range appList {
		result[app.ID] = app.Name
	}
	return result, nil
}
