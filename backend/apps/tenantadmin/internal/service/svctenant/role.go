package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
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

// Create 创建租户角色（编码租户内唯一）。
func (svc *roleSvc) Create(ctx *gin.Context, req *dtotenant.RoleCreateReq) (*dtotenant.RoleCreateResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	if req.Type == "" {
		req.Type = "User"
	}

	// 角色从属于租户订阅的应用：校验 appID
	appList, err := loadTenantApps(ctx)
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

	// 编码应用内唯一
	existing, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, AppID: req.AppID, Code: req.Code})
	if err != nil {
		glog.Errorf(ctx, "[svcrole.Create] query role by code fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleCreateError)
	}
	if len(existing) > 0 {
		return nil, code.GetError(code.RoleCreateError)
	}

	insertEntity := &model.RoleEntity{
		TenantID:    tenantID,
		AppID:       req.AppID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Type:        req.Type,
		Source:      string(model.RoleSourceCustom),
		AdminLevel:  string(model.SysAdminLevelNone),
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
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil || !roleVisibleToTenant(roleEntity, tenantID) {
		return code.GetError(code.RoleNotExistError)
	}
	// 内置角色禁止删除（防止系统管理能力失控且 seed 幂等不自动重建）
	if roleEntity.Source == string(model.RoleSourceBuiltin) {
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
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil || !roleVisibleToTenant(roleEntity, tenantID) {
		return code.GetError(code.RoleNotExistError)
	}

	updateMap := map[string]any{
		"name":        req.Name,
		"code":        req.Code,
		"description": req.Description,
		"type":        req.Type,
		"updated_by":  gincontext.GetUserIDString(ctx),
	}
	// 内置角色保护：禁止改核心字段（编码/类别），名称与描述仍可改
	if roleEntity.Source == string(model.RoleSourceBuiltin) {
		if req.Code != roleEntity.Code || req.Type != roleEntity.Type {
			return code.GetError(code.RoleUpdateBuiltinForbiddenError)
		}
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
	if err != nil || !roleVisibleToTenant(roleEntity, tenantID) {
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
		Name:        roleEntity.Name,
		Code:        roleEntity.Code,
		Description: roleEntity.Description,
		Type:        roleEntity.Type,
		Source:      roleEntity.Source,
		AdminLevel:  roleEntity.AdminLevel,
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
		TenantID: tenantID,
		AppID:    req.AppID,
		Keyword:  req.Keyword,
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
			Name:        v.Name,
			Code:        v.Code,
			Description: v.Description,
			Type:        v.Type,
			Source:      v.Source,
			AdminLevel:  v.AdminLevel,
			MemberCount: memberCount[v.ID],
			MenuCount:   menuCount[v.ID],
			CreatedAt:   v.CreatedAt.Unix(),
		})
	}
	return &dtotenant.RolePageListResp{List: list, Total: total}, nil
}

// GetMenus 角色菜单授权回显：角色所属应用的菜单树 + 已授权菜单ID（无应用归属的种子角色回退全控制台菜单）。
func (svc *roleSvc) GetMenus(ctx *gin.Context, req *dtotenant.RoleDetailReq) (*dtotenant.RoleMenuTreeResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil || !roleVisibleToTenant(roleEntity, tenantID) {
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

// roleMenuTree 角色可授权的菜单树：有应用归属则取该应用菜单，否则回退全租户控制台菜单（种子角色）。
func (svc *roleSvc) roleMenuTree(ctx *gin.Context, roleEntity *model.RoleEntity) ([]dtotenant.MenuTreeItem, error) {
	if roleEntity.AppID != "" {
		return buildAppMenuTree(ctx, roleEntity.AppID)
	}
	return buildTenantMenuTree(ctx)
}

// UpdateMenus 全量替换角色菜单授权（PUT 集合语义）。
func (svc *roleSvc) UpdateMenus(ctx *gin.Context, req *dtotenant.RoleMenusUpdateReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil || !roleVisibleToTenant(roleEntity, tenantID) {
		return code.GetError(code.RoleNotExistError)
	}

	// 校验菜单均属于角色所属应用（无应用归属的种子角色校验全租户控制台菜单）
	if len(req.MenuIDs) > 0 {
		tree, err := svc.roleMenuTree(ctx, roleEntity)
		if err != nil {
			return err
		}
		allowed := collectMenuIDs(tree)
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

// tenantAppNameMap 租户订阅应用 ID -> 名称 映射。
func tenantAppNameMap(ctx *gin.Context) (map[string]string, error) {
	appList, err := loadTenantApps(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(appList))
	for _, app := range appList {
		result[app.ID] = app.Name
	}
	return result, nil
}
