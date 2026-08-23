package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
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
		if err != nil {
			glog.Errorf(ctx, "[svctenant.loadTenantApps] application GetByID fail, err:%v, appID:%s", err, item.AppID)
			continue
		}
		if appEntity == nil || appEntity.ID == "" {
			glog.Warnf(ctx, "[svctenant.loadTenantApps] application not exist, appID:%s", item.AppID)
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
		Status: model.MenuStatusEnable,
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
//   - 内置管理员豁免：持有内置管理员角色（source=builtin && admin_level=super）→ 全量菜单（含 visibility=admin，免授权）；
//   - 普通成员：按该用户授权菜单集合（role_menu 并集）过滤 + visibility 门槛（public/member）二次过滤；
//     父子收敛：父未达标/未授权时若存在可见子项则保留父壳，保证层级连贯。
func buildMyMenuTree(ctx *gin.Context) ([]dtotenant.MenuTreeItem, error) {
	full, err := buildTenantMenuTree(ctx)
	if err != nil {
		return nil, err
	}
	admin, err := userHoldsBuiltinAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if admin {
		return full, nil
	}
	authed, err := userAuthorizedMenuIDs(ctx)
	if err != nil {
		return nil, err
	}
	return pruneMenuTreeByAuthed(full, authed, model.MenuVisibilityMember.VisibilityRank()), nil
}

// userHoldsBuiltinAdmin 判断当前用户是否持有内置管理员角色（source=builtin && admin_level=super）。
func userHoldsBuiltinAdmin(ctx *gin.Context) (bool, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userID := gincontext.GetUserIDString(ctx)
	if tenantID == "" || userID == "" {
		return false, nil
	}
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.userHoldsBuiltinAdmin] query user_role fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
		return false, err
	}
	if len(urList) == 0 {
		return false, nil
	}
	roleIDs := make([]string, 0, len(urList))
	for _, r := range urList {
		roleIDs = append(roleIDs, r.RoleID)
	}
	roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.userHoldsBuiltinAdmin] query role fail, err:%v, tenantID:%s", err, tenantID)
		return false, err
	}
	for i := range roleList {
		r := &roleList[i]
		if r.IsBuiltinAdmin() {
			return true, nil
		}
	}
	return false, nil
}

// userAuthorizedMenuIDs 返回当前用户在指定租户下经 role_menu 授权得到的菜单 ID 集合（跨角色去重）。
func userAuthorizedMenuIDs(ctx *gin.Context) (map[string]bool, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userID := gincontext.GetUserIDString(ctx)
	result := make(map[string]bool)
	if tenantID == "" || userID == "" {
		return result, nil
	}
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.userAuthorizedMenuIDs] query user_role fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
		return nil, err
	}
	if len(urList) == 0 {
		return result, nil
	}
	roleIDs := make([]string, 0, len(urList))
	for _, r := range urList {
		roleIDs = append(roleIDs, r.RoleID)
	}
	rmList, err := dao.NewRoleMenuDao().GetListByCond(ctx, &dao.RoleMenuCond{TenantID: tenantID, RoleIDs: roleIDs})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.userAuthorizedMenuIDs] query role_menu fail, err:%v, tenantID:%s", err, tenantID)
		return nil, err
	}
	for _, rm := range rmList {
		result[rm.MenuID] = true
	}
	return result, nil
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

// pruneMenuTreeByAuthed 按「授权集合 + 可见等级」剪枝菜单树：菜单需同时命中已授权且可见等级达标；
// 父未命中但存在命中子项时保留父壳（导航分组），保证层级连贯。
func pruneMenuTreeByAuthed(items []dtotenant.MenuTreeItem, authed map[string]bool, level int) []dtotenant.MenuTreeItem {
	result := make([]dtotenant.MenuTreeItem, 0, len(items))
	for _, item := range items {
		children := pruneMenuTreeByAuthed(item.Children, authed, level)
		itemVis := model.MenuVisibility(item.Visibility).VisibilityRank()
		if itemVis <= level && authed[item.MenuID] {
			item.Children = children
			result = append(result, item)
			continue
		}
		// 父未达标/未授权：仅当有命中子菜单时保留父壳（作为导航分组）
		if len(children) > 0 {
			item.Children = children
			result = append(result, item)
		}
	}
	return result
}

// HasSystemAdminCapability 判断当前用户（按 gin 上下文取租户/用户）是否具备「系统管理能力」
// （admin_level == super）。授权驱动：聚合该用户全部角色取最高 admin_level。
func HasSystemAdminCapability(ctx *gin.Context) (bool, error) {
	level, err := ResolveUserAdminLevel(ctx)
	if err != nil {
		return false, err
	}
	return level.HasSystemAdmin(), nil
}

// ResolveUserAdminLevel 推导当前用户能达到的最高系统管理等级：聚合该用户全部角色，
// 取各角色 admin_level（显式能力标签）的最高档位（member < super）。
func ResolveUserAdminLevel(ctx *gin.Context) (model.SysAdminLevel, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userID := gincontext.GetUserIDString(ctx)
	if tenantID == "" || userID == "" {
		return model.SysAdminLevelMember, nil
	}
	// 1. 用户 → 角色
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.ResolveUserAdminLevel] query user_role fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
		return model.SysAdminLevelMember, err
	}
	if len(urList) == 0 {
		return model.SysAdminLevelMember, nil
	}
	roleIDs := make([]string, 0, len(urList))
	for _, r := range urList {
		roleIDs = append(roleIDs, r.RoleID)
	}
	// 2. 角色 → 取最高 admin_level
	roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.ResolveUserAdminLevel] query role fail, err:%v, tenantID:%s", err, tenantID)
		return model.SysAdminLevelMember, err
	}
	level := model.SysAdminLevelMember
	for i := range roleList {
		lv := model.SysAdminLevel(roleList[i].AdminLevel)
		if lv.SysAdminRank() > level.SysAdminRank() {
			level = lv
		}
	}
	return level, nil
}
