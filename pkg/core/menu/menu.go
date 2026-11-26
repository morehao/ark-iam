// Package menu 提供跨应用（tenantadmin / platformadmin）复用的菜单可见性逻辑：
//   - 构建指定应用的启用菜单树；
//   - 判断用户是否持有内置管理员角色（管理员角色豁免全量）；
//   - 计算用户在租户内的角色授权菜单集合；
//   - 按「授权集合 + 可见性门槛」剪枝菜单树。
//
// 本包只依赖 pkg/{dao,model,object}，不引用任何具体应用，供两端侧边栏动态菜单共用。
package menu

import (
	"context"
	"errors"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

// ErrParentUnusable 是菜单父级「业务性不可用」的哨兵错误：父菜单不存在（nil/空 ID）
// 或不属于当前应用（parent.AppID != appID）。调用方据此与不可预期的系统错误分离——
// 系统错误（DAO 失败）由各函数原样返回并已记日志，业务性不可用返回本哨兵。
var ErrParentUnusable = errors.New("menu parent unusable")

// BuildAppMenuTree 构建指定应用的启用菜单树（角色菜单授权 / 侧边栏可见树共用）。
// 展示顺序取 dao.MenuOrderBySort：同一父级下按控制台维护的「排序」升序——
// 侧边栏与角色授权树的顺序必须是「排序」字段的唯一函数，否则控制台里改排序看不到任何效果。
func BuildAppMenuTree(ctx context.Context, appID string) ([]objpermission.MenuItemNode, error) {
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

	// 先按 parent_id 建一次索引（保留 dao 的排序顺序），再递归组装，整体 O(n)。
	// 旧实现对每个节点全量扫描查找子节点，是 O(n²)，控制台每次加载侧边栏 × 应用数会放大。
	childrenByParent := make(map[string][]*model.MenuEntity, len(menus))
	for _, menu := range menus {
		childrenByParent[menu.ParentID] = append(childrenByParent[menu.ParentID], menu)
	}

	var buildTree func(parentID string) []objpermission.MenuItemNode
	buildTree = func(parentID string) []objpermission.MenuItemNode {
		children := childrenByParent[parentID]
		if len(children) == 0 {
			// 与旧实现保持一致：无子节点返回空切片（非 nil），避免 JSON 序列化为 null。
			return []objpermission.MenuItemNode{}
		}
		items := make([]objpermission.MenuItemNode, 0, len(children))
		for _, menu := range children {
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
func BuildMyMenuTree(ctx context.Context, tenantID, userID string, appIDs []string) ([]objpermission.MenuItemNode, error) {
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
func UserHoldsBuiltinAdmin(ctx context.Context, tenantID, userID string) (bool, error) {
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
func UserAuthorizedMenuIDs(ctx context.Context, tenantID, userID string) (map[string]bool, error) {
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
func ResolveUserAdminType(ctx context.Context, tenantID, userID string) (model.SysAdminType, error) {
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

// ValidateMenuEnums 校验菜单字典枚举合法值：Type/Status/Visibility 必须命中白名单常量；
// Hidden/ExternalLink/KeepAlive 三个开关允许空串（＝调用方未提供，交由 NormalizeMenuSwitches 归一为 disable）。
func ValidateMenuEnums(req *objpermission.MenuBaseInfo) bool {
	switch req.Type {
	case model.MenuTypeDirectory, model.MenuTypeMenu, model.MenuTypeButton:
	default:
		return false
	}
	switch req.Status {
	case model.MenuStatusEnable, model.MenuStatusDisable:
	default:
		return false
	}
	switch req.Visibility {
	case model.MenuVisibilityPublic, model.MenuVisibilityMember, model.MenuVisibilityAdmin:
	default:
		return false
	}
	switch req.Hidden {
	case "", model.MenuHiddenFlagEnable, model.MenuHiddenFlagDisable:
	default:
		return false
	}
	switch req.ExternalLink {
	case "", model.MenuExternalLinkFlagEnable, model.MenuExternalLinkFlagDisable:
	default:
		return false
	}
	switch req.KeepAlive {
	case "", model.MenuKeepAliveFlagEnable, model.MenuKeepAliveFlagDisable:
	default:
		return false
	}
	return true
}

// NormalizeMenuSwitches 把未提供的开关（空串）归一为 disable：列默认值即 disable，
// 而更新走 map（不经过 GORM 的默认值省略），空串直写会落成脏枚举值；
// 归一后语义与原 bool 字段的 false（未勾选）完全一致。
func NormalizeMenuSwitches(req *objpermission.MenuBaseInfo) {
	if req.Hidden == "" {
		req.Hidden = model.MenuHiddenFlagDisable
	}
	if req.ExternalLink == "" {
		req.ExternalLink = model.MenuExternalLinkFlagDisable
	}
	if req.KeepAlive == "" {
		req.KeepAlive = model.MenuKeepAliveFlagDisable
	}
}

// MenuParentUsable 校验上级菜单并返回父菜单实体：空串 parentID 表示根菜单（合法，返回 (nil, nil)）；
// 否则父级必须存在且与本菜单同属一个应用。返回 ErrParentUnusable 表示业务性不可用
// （父不存在/不属于该应用），系统错误（DAO 失败）原样返回。区分两者是既有 bool 返回值掩盖的瑕疵：
// 调用方据此可分别返回「不存在」与「操作失败」两类错误码。
// 不校验会让子菜单挂到不存在或别的应用的父级下——它不会出现在任何应用菜单树里，成为看不见的孤儿行。
func MenuParentUsable(db gormdao.DBGetter, ctx context.Context, appID, parentID string) (*model.MenuEntity, error) {
	if parentID == "" {
		return nil, nil
	}
	parent, err := dao.NewMenuDao(dao.WithDBGetter(db)).GetByID(ctx, parentID)
	if err != nil {
		glog.Errorf(ctx, "[menu.MenuParentUsable] dao GetByID fail, err:%v, parentID:%s", err, parentID)
		return nil, err
	}
	if parent == nil || parent.ID == "" || parent.AppID != appID {
		return nil, ErrParentUnusable
	}
	return parent, nil
}

// CollectSubtreeIDs 返回 rootID 及其全部子孙菜单 ID（限定同一应用）。
// 菜单树规模有限（单应用几十行），一次取回后内存遍历，避免逐层查询。
func CollectSubtreeIDs(db gormdao.DBGetter, ctx context.Context, appID, rootID string) ([]string, error) {
	list, err := dao.NewMenuDao(dao.WithDBGetter(db)).GetListByCond(ctx, &dao.MenuCond{AppID: appID})
	if err != nil {
		glog.Errorf(ctx, "[menu.CollectSubtreeIDs] dao GetListByCond fail, err:%v, appID:%s", err, appID)
		return nil, err
	}
	childrenByParent := make(map[string][]string, len(list))
	for _, item := range list {
		childrenByParent[item.ParentID] = append(childrenByParent[item.ParentID], item.ID)
	}
	ids := make([]string, 0, 8)
	queue := []string{rootID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ids = append(ids, id)
		queue = append(queue, childrenByParent[id]...)
	}
	return ids, nil
}
