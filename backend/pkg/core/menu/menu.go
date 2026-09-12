// Package menu 提供跨应用（tenantadmin / platformadmin）复用的菜单可见性逻辑：
//   - 构建指定应用的启用菜单树；
//   - 判断用户是否持有内置管理员角色（管理员角色豁免全量）；
//   - 计算用户在租户内的角色授权菜单集合；
//   - 按「授权集合 + 可见性门槛」剪枝菜单树。
//
// 本包只依赖 pkg/{dao,model,object}，不引用任何具体应用，供两端侧边栏动态菜单共用。
package menu

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

// BuildAppMenuTree 构建指定应用的启用菜单树（角色菜单授权 / 侧边栏可见树共用）。
// 展示顺序取 dao.MenuOrderBySort：同一父级下按控制台维护的「排序」升序——
// 侧边栏与角色授权树的顺序必须是「排序」字段的唯一函数，否则控制台里改排序看不到任何效果。
func BuildAppMenuTree(ctx *gin.Context, appID string) ([]objpermission.MenuItemNode, error) {
	menuEntityList, _, err := dao.NewMenuDao().GetPageListByCond(ctx, &dao.MenuCond{
		BaseCond: &gormdao.BaseCond{OrderField: dao.MenuOrderBySort},
		AppID:    appID,
		Status:   model.MenuStatusEnable,
	})
	if err != nil {
		glog.Errorf(ctx, "[menu.BuildAppMenuTree] dao menu GetPageListByCond fail, err:%v, appID:%s", err, appID)
		return nil, err
	}
	menus := make([]*model.MenuEntity, 0, len(menuEntityList))
	for i := range menuEntityList {
		menus = append(menus, &menuEntityList[i])
	}

	var buildTree func(parentID string) []objpermission.MenuItemNode
	buildTree = func(parentID string) []objpermission.MenuItemNode {
		var items []objpermission.MenuItemNode
		for _, menu := range menus {
			if menu.ParentID != parentID {
				continue
			}
			item := objpermission.MenuItemNode{
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

// BuildMyMenuTree 构建当前用户在指定应用范围内可见的菜单树：
//   - 内置管理员豁免：持有内置管理员角色（source=builtin && admin_type=admin）→ 全量菜单（含 visibility=admin，免授权）；
//   - 普通用户：按该用户授权菜单集合（role_menu 并集）过滤 + visibility 门槛（public/member）二次过滤；
//     父子收敛：父未达标/未授权时若存在可见子项则保留父壳，保证层级连贯。
func BuildMyMenuTree(ctx *gin.Context, tenantID, userID string, appIDs []string) ([]objpermission.MenuItemNode, error) {
	var full []objpermission.MenuItemNode
	for _, appID := range appIDs {
		appTree, err := BuildAppMenuTree(ctx, appID)
		if err != nil {
			return nil, err
		}
		full = append(full, appTree...)
	}

	admin, err := UserHoldsBuiltinAdmin(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if admin {
		return full, nil
	}
	authed, err := UserAuthorizedMenuIDs(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return PruneMenuTreeByAuthed(full, authed, model.MenuVisibilityMember.VisibilityRank()), nil
}

// UserHoldsBuiltinAdmin 判断用户（租户内）是否持有内置管理员角色（source=builtin && admin_type=admin）。
func UserHoldsBuiltinAdmin(ctx *gin.Context, tenantID, userID string) (bool, error) {
	if tenantID == "" || userID == "" {
		return false, nil
	}
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[menu.UserHoldsBuiltinAdmin] query user_role fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
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
		glog.Errorf(ctx, "[menu.UserHoldsBuiltinAdmin] query role fail, err:%v, tenantID:%s", err, tenantID)
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

// UserAuthorizedMenuIDs 返回用户在租户内经 role_menu 授权得到的菜单 ID 集合（跨角色去重）。
func UserAuthorizedMenuIDs(ctx *gin.Context, tenantID, userID string) (map[string]bool, error) {
	result := make(map[string]bool)
	if tenantID == "" || userID == "" {
		return result, nil
	}
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[menu.UserAuthorizedMenuIDs] query user_role fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
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
		glog.Errorf(ctx, "[menu.UserAuthorizedMenuIDs] query role_menu fail, err:%v, tenantID:%s", err, tenantID)
		return nil, err
	}
	for _, rm := range rmList {
		result[rm.MenuID] = true
	}
	return result, nil
}

// ResolveUserAdminType 推导用户的系统管理类型：聚合其全部角色，
// 任一角色为管理员类型（admin）即视为管理员类型，否则为普通类型（normal）。
func ResolveUserAdminType(ctx *gin.Context, tenantID, userID string) (model.SysAdminType, error) {
	if tenantID == "" || userID == "" {
		return model.SysAdminTypeNormal, nil
	}
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[menu.ResolveUserAdminType] query user_role fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
		return model.SysAdminTypeNormal, err
	}
	if len(urList) == 0 {
		return model.SysAdminTypeNormal, nil
	}
	roleIDs := make([]string, 0, len(urList))
	for _, r := range urList {
		roleIDs = append(roleIDs, r.RoleID)
	}
	roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		glog.Errorf(ctx, "[menu.ResolveUserAdminType] query role fail, err:%v, tenantID:%s", err, tenantID)
		return model.SysAdminTypeNormal, err
	}
	for i := range roleList {
		if roleList[i].AdminType.HasSystemAdmin() {
			return model.SysAdminTypeAdmin, nil
		}
	}
	return model.SysAdminTypeNormal, nil
}

// PruneMenuTree 按可见等级剪枝菜单树：可见(保留下钻) 或略过，父菜单达标则递归保留子树。
func PruneMenuTree(items []objpermission.MenuItemNode, level int) []objpermission.MenuItemNode {
	result := make([]objpermission.MenuItemNode, 0, len(items))
	for _, item := range items {
		itemVis := item.Visibility.VisibilityRank()
		children := PruneMenuTree(item.Children, level)
		if itemVis <= level {
			item.Children = children
			result = append(result, item)
			continue
		}
		if len(children) > 0 {
			item.Children = children
			result = append(result, item)
		}
	}
	return result
}

// PruneMenuTreeByAuthed 按「授权集合 + 可见等级」剪枝菜单树：菜单需同时命中已授权且可见等级达标；
// 父未命中但存在命中子项时保留父壳（导航分组），保证层级连贯。
func PruneMenuTreeByAuthed(items []objpermission.MenuItemNode, authed map[string]bool, level int) []objpermission.MenuItemNode {
	result := make([]objpermission.MenuItemNode, 0, len(items))
	for _, item := range items {
		children := PruneMenuTreeByAuthed(item.Children, authed, level)
		itemVis := item.Visibility.VisibilityRank()
		if itemVis <= level && authed[item.MenuID] {
			item.Children = children
			result = append(result, item)
			continue
		}
		if len(children) > 0 {
			item.Children = children
			result = append(result, item)
		}
	}
	return result
}
