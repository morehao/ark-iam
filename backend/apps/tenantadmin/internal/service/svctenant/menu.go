package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

// TenantMenuSvc 租户侧菜单服务
type TenantMenuSvc interface {
	Tree(ctx *gin.Context) (*dtotenant.MenuTreeResp, error)
	Apps(ctx *gin.Context) (*dtotenant.TenantAppsResp, error)
}

type tenantMenuSvc struct{}

var _ TenantMenuSvc = (*tenantMenuSvc)(nil)

func NewTenantMenuSvc() TenantMenuSvc {
	return &tenantMenuSvc{}
}

func (svc *tenantMenuSvc) Tree(ctx *gin.Context) (*dtotenant.MenuTreeResp, error) {
	tree, err := buildMyMenuTree(ctx)
	if err != nil {
		return nil, err
	}
	return &dtotenant.MenuTreeResp{
		List: tree,
	}, nil
}

// Apps 当前租户订阅的启用非系统应用（角色归属/菜单授权的应用选项）。
func (svc *tenantMenuSvc) Apps(ctx *gin.Context) (*dtotenant.TenantAppsResp, error) {
	appList, err := loadTenantApps(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]dtotenant.TenantAppItem, 0, len(appList))
	for _, app := range appList {
		list = append(list, dtotenant.TenantAppItem{
			AppID: app.ID,
			Code:  app.Code,
			Name:  app.Name,
		})
	}
	return &dtotenant.TenantAppsResp{List: list}, nil
}

// loadTenantApps 当前租户订阅的启用非系统应用（排除平台系统内置应用，如管理后台）。
func loadTenantApps(ctx *gin.Context) ([]model.ApplicationEntity, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	tenantAppList, _, err := dao.NewTenantApplicationDao().GetPageListByCond(ctx, &dao.TenantApplicationCond{
		TenantID: tenantID,
		Status:   model.AppStatusEnable,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.loadTenantApps] dao tenantApplication GetPageListByCond fail, err:%v, tenantID:%s", err, tenantID)
		return nil, code.GetError(code.MenuGetPageListError)
	}

	appIDSet := make(map[string]struct{})
	appIDs := make([]string, 0, len(tenantAppList))
	for _, item := range tenantAppList {
		if item.AppID == "" {
			continue
		}
		if _, ok := appIDSet[item.AppID]; ok {
			continue
		}
		appEntity, err := dao.NewApplicationDao().GetByID(ctx, item.AppID)
		if err != nil || appEntity == nil || appEntity.ID == "" {
			glog.Warnf(ctx, "[svctenant.loadTenantApps] application GetByID fail or not exist, err:%v, appID:%s", err, item.AppID)
			continue
		}
		if appEntity.IsSystem {
			continue
		}
		appIDSet[item.AppID] = struct{}{}
		appIDs = append(appIDs, item.AppID)
	}
	if len(appIDs) == 0 {
		return nil, nil
	}
	appList, err := dao.NewApplicationDao().GetListByCond(ctx, &dao.ApplicationCond{IDs: appIDs})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.loadTenantApps] dao application GetListByCond fail, err:%v", err)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	return appList, nil
}

// buildAppMenuTree 构建指定应用的启用菜单树（角色菜单授权用）。
func buildAppMenuTree(ctx *gin.Context, appID string) ([]dtotenant.MenuTreeItem, error) {
	menuEntityList, _, err := dao.NewMenuDao().GetPageListByCond(ctx, &dao.MenuCond{
		AppID:  appID,
		Status: model.AppStatusEnable,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.buildAppMenuTree] dao menu GetPageListByCond fail, err:%v, appID:%s", err, appID)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	menus := make([]*model.MenuEntity, 0, len(menuEntityList))
	for i := range menuEntityList {
		menus = append(menus, &menuEntityList[i])
	}

	var buildTree func(parentID string) []dtotenant.MenuTreeItem
	buildTree = func(parentID string) []dtotenant.MenuTreeItem {
		var items []dtotenant.MenuTreeItem
		for _, menu := range menus {
			if menu.ParentID != parentID {
				continue
			}
			item := dtotenant.MenuTreeItem{
				MenuID: menu.ID,
				MenuBaseInfo: objpermission.MenuBaseInfo{
					AppID:        menu.AppID,
					ParentID:     menu.ParentID,
					Name:         menu.Name,
					Code:         menu.Code,
					Path:         menu.Path,
					Icon:         menu.Icon,
					Sort:         menu.Sort,
					Type:         menu.Type,
					Visibility:   menu.Visibility,
					Component:    menu.Component,
					Redirect:     menu.Redirect,
					Hidden:       menu.Hidden,
					ExternalLink: menu.ExternalLink,
					KeepAlive:    menu.KeepAlive,
					Permission:   menu.Permission,
					Status:       menu.Status,
				},
				Children: buildTree(menu.ID),
			}
			items = append(items, item)
		}
		return items
	}
	return buildTree(""), nil
}

// buildTenantMenuTree 构建租户控制台菜单树（全部订阅的非系统应用），供侧边栏使用。
func buildTenantMenuTree(ctx *gin.Context) ([]dtotenant.MenuTreeItem, error) {
	appList, err := loadTenantApps(ctx)
	if err != nil {
		return nil, err
	}
	var tree []dtotenant.MenuTreeItem
	for _, app := range appList {
		appTree, err := buildAppMenuTree(ctx, app.ID)
		if err != nil {
			return nil, err
		}
		tree = append(tree, appTree...)
	}
	return tree, nil
}

// buildMyMenuTree 构建当前用户可见的租户控制台菜单树：
// 在 buildTenantMenuTree 的全量菜单基础上，按「可见性门槛（visibility）<= 当前用户可达等级」过滤；
// 父子收敛：父菜单达标则整棵保留（含其全部子菜单），避免出现子可见而父不可见的不连贯结构。
func buildMyMenuTree(ctx *gin.Context) ([]dtotenant.MenuTreeItem, error) {
	level, err := resolveUserMenuLevel(ctx)
	if err != nil {
		return nil, err
	}
	full, err := buildTenantMenuTree(ctx)
	if err != nil {
		return nil, err
	}
	return pruneMenuTree(full, level), nil
}

// resolveUserMenuLevel 计算当前用户能达到的最高可见档位（rank）。租户/tenant 端点本身要求已登录成员，
// 故基线为 member；若具备系统管理能力则升为 admin。
func resolveUserMenuLevel(ctx *gin.Context) (int, error) {
	hasAdmin, err := HasSystemAdminCapability(ctx)
	if err != nil {
		return model.MenuVisibilityMember.VisibilityRank(), nil
	}
	if hasAdmin {
		return model.MenuVisibilityAdmin.VisibilityRank(), nil
	}
	return model.MenuVisibilityMember.VisibilityRank(), nil
}

// pruneMenuTree 按可见等级剪枝菜单树：可见(保留下钻) 或略过，父菜单达标则递归保留子树。
func pruneMenuTree(items []dtotenant.MenuTreeItem, level int) []dtotenant.MenuTreeItem {
	result := make([]dtotenant.MenuTreeItem, 0, len(items))
	for _, item := range items {
		itemVis := model.MenuVisibility(item.Visibility).VisibilityRank()
		// 子菜单：先剪子树；父菜单即使不达标，若其可见子项存在则保留父壳，保证层级连贯
		children := pruneMenuTree(item.Children, level)
		if itemVis <= level {
			item.Children = children
			result = append(result, item)
			continue
		}
		// 父不达标：仅当有可见子菜单时保留父壳（作为导航分组）
		if len(children) > 0 {
			item.Children = children
			result = append(result, item)
		}
	}
	return result
}

// HasSystemAdminCapability 判断当前用户（按 gin 上下文取租户/用户）是否具备「系统管理能力」。
// 授权驱动：遍历该用户的全部角色，聚合其 scope，推导系统管理等级（>= basic 即视为具备系统管理能力）。
// 说明：此处以「拥有管理资源 scope」作为判定，而非角色 type 硬编码；与 OIDC token 的 scope 口径一致。
func HasSystemAdminCapability(ctx *gin.Context) (bool, error) {
	level, err := ResolveUserAdminLevel(ctx)
	if err != nil {
		return false, err
	}
	return level.HasSystemAdmin(), nil
}

// ResolveUserAdminLevel 推导当前用户能达到的最高系统管理等级（授权驱动，聚合该用户全部角色的 scope）。
func ResolveUserAdminLevel(ctx *gin.Context) (model.SysAdminLevel, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userID := gincontext.GetUserIDString(ctx)
	if tenantID == "" || userID == "" {
		return model.SysAdminLevelNone, nil
	}
	// 1. 用户 → 角色
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.ResolveUserAdminLevel] query user_role fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
		return model.SysAdminLevelNone, err
	}
	if len(urList) == 0 {
		return model.SysAdminLevelNone, nil
	}
	roleIDs := make([]string, 0, len(urList))
	for _, r := range urList {
		roleIDs = append(roleIDs, r.RoleID)
	}
	// 2. 角色 → scope 关联
	var rsList []model.RoleScopeEntity
	if err := gormDBFromCtx(ctx).Where("tenant_id = ? AND role_id IN ?", tenantID, roleIDs).Find(&rsList).Error; err != nil {
		glog.Errorf(ctx, "[svctenant.ResolveUserAdminLevel] query role_scope fail, err:%v", err)
		return model.SysAdminLevelNone, err
	}
	if len(rsList) == 0 {
		return model.SysAdminLevelNone, nil
	}
	scopeIDs := make([]string, 0, len(rsList))
	for _, rs := range rsList {
		scopeIDs = append(scopeIDs, rs.ScopeID)
	}
	// 3. scope → 聚合名称并推导等级
	var scopeList []model.ScopeEntity
	if err := gormDBFromCtx(ctx).Where("tenant_id = ? AND id IN ?", tenantID, scopeIDs).Find(&scopeList).Error; err != nil {
		glog.Errorf(ctx, "[svctenant.ResolveUserAdminLevel] query scope fail, err:%v", err)
		return model.SysAdminLevelNone, err
	}
	names := make([]string, 0, len(scopeList))
	for _, s := range scopeList {
		names = append(names, s.Name)
	}
	return model.DeriveAdminLevelFromScopeNames(names), nil
}

// SyncRoleAdminLevel 推导弹角色 admin_level 并回写（授权驱动投影同步入口）。
// 供角色 scope 授权变更后调用：加载该角色已授 scope → 推导等级 → 若与当前不一致则更新列。
func SyncRoleAdminLevel(ctx *gin.Context, role *model.RoleEntity) error {
	if role == nil || role.ID == "" {
		return nil
	}
	if role.AdminLevel == "" {
		role.AdminLevel = string(model.SysAdminLevelNone)
	}
	// 该角色已授 scope 名称
	var rsList []model.RoleScopeEntity
	if err := gormDBFromCtx(ctx).Where("tenant_id = ? AND role_id = ?", role.TenantID, role.ID).Find(&rsList).Error; err != nil {
		glog.Errorf(ctx, "[svctenant.SyncRoleAdminLevel] query role_scope fail, err:%v, roleID:%s", err, role.ID)
		return err
	}
	var scopeIDs []string
	for _, rs := range rsList {
		scopeIDs = append(scopeIDs, rs.ScopeID)
	}
	names := []string{}
	if len(scopeIDs) > 0 {
		var scopeList []model.ScopeEntity
		if err := gormDBFromCtx(ctx).Where("tenant_id = ? AND id IN ?", role.TenantID, scopeIDs).Find(&scopeList).Error; err != nil {
			glog.Errorf(ctx, "[svctenant.SyncRoleAdminLevel] query scope fail, err:%v, roleID:%s", err, role.ID)
			return err
		}
		for _, s := range scopeList {
			names = append(names, s.Name)
		}
	}
	target := string(model.DeriveAdminLevelFromScopeNames(names))
	if role.AdminLevel == target {
		return nil
	}
	if err := gormDBFromCtx(ctx).Model(&model.RoleEntity{}).Where("id = ?", role.ID).
		UpdateColumn("admin_level", target).Error; err != nil {
		glog.Errorf(ctx, "[svctenant.SyncRoleAdminLevel] update admin_level fail, err:%v, roleID:%s", err, role.ID)
		return err
	}
	role.AdminLevel = target
	return nil
}

// gormDBFromCtx 取得公共 IAM 库的 *gorm.DB（封装 dbclient，供本次聚合查询使用）。
func gormDBFromCtx(ctx *gin.Context) *gorm.DB {
	return dbclient.IamDB(ctx)
}
