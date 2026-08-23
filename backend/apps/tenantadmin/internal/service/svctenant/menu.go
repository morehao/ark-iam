package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/pkg/iam/svcmenu"
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

// buildAppMenuTree 构建指定应用的启用菜单树（角色菜单授权用），基于公共层 svcmenu.BuildAppMenuTree。
func buildAppMenuTree(ctx *gin.Context, appID string) ([]dtotenant.MenuTreeItem, error) {
	nodes, err := svcmenu.BuildAppMenuTree(ctx, appID)
	if err != nil {
		return nil, code.GetError(code.MenuGetPageListError)
	}
	return convertMenuNodes(nodes), nil
}

// convertMenuNodes 把公共层菜单节点转换为租户侧 DTO 菜单树。
func convertMenuNodes(nodes []objpermission.MenuItemNode) []dtotenant.MenuTreeItem {
	result := make([]dtotenant.MenuTreeItem, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, dtotenant.MenuTreeItem{
			MenuID:       node.MenuID,
			MenuBaseInfo: node.MenuBaseInfo,
			Children:     convertMenuNodes(node.Children),
		})
	}
	return result
}

// toMenuNodes 把租户侧 DTO 菜单树转换为公共层菜单节点（剪枝等仅关心 menuID/visibility/children）。
func toMenuNodes(items []dtotenant.MenuTreeItem) []objpermission.MenuItemNode {
	result := make([]objpermission.MenuItemNode, 0, len(items))
	for _, item := range items {
		result = append(result, objpermission.MenuItemNode{
			MenuID:       item.MenuID,
			MenuBaseInfo: item.MenuBaseInfo,
			Children:     toMenuNodes(item.Children),
		})
	}
	return result
}

// pruneMenuTree 按可见等级剪枝菜单树（适配租户侧 DTO，复用公共层 svcmenu）。
func pruneMenuTree(items []dtotenant.MenuTreeItem, level int) []dtotenant.MenuTreeItem {
	return convertMenuNodes(svcmenu.PruneMenuTree(toMenuNodes(items), level))
}

// pruneMenuTreeByAuthed 按「授权集合 + 可见等级」剪枝菜单树（适配租户侧 DTO，复用公共层 svcmenu）。
func pruneMenuTreeByAuthed(items []dtotenant.MenuTreeItem, authed map[string]bool, level int) []dtotenant.MenuTreeItem {
	return convertMenuNodes(svcmenu.PruneMenuTreeByAuthed(toMenuNodes(items), authed, level))
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
	tenantID := gincontext.GetTenantIDString(ctx)
	userID := gincontext.GetUserIDString(ctx)
	appList, err := loadTenantApps(ctx)
	if err != nil {
		return nil, err
	}
	appIDs := make([]string, 0, len(appList))
	for _, app := range appList {
		appIDs = append(appIDs, app.ID)
	}
	nodes, err := svcmenu.BuildMyMenuTree(ctx, tenantID, userID, appIDs)
	if err != nil {
		return nil, err
	}
	return convertMenuNodes(nodes), nil
}

// userHoldsBuiltinAdmin 判断当前用户是否持有内置管理员角色（source=builtin && admin_level=super）。
// 保留薄包装以复用公共层实现并维持既有测试契约。
func userHoldsBuiltinAdmin(ctx *gin.Context) (bool, error) {
	return svcmenu.UserHoldsBuiltinAdmin(ctx, gincontext.GetTenantIDString(ctx), gincontext.GetUserIDString(ctx))
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
// 取各角色 admin_level（显式能力标签）的最高档位（member < super）。复用公共层 svcmenu。
func ResolveUserAdminLevel(ctx *gin.Context) (model.SysAdminLevel, error) {
	return svcmenu.ResolveUserAdminLevel(ctx, gincontext.GetTenantIDString(ctx), gincontext.GetUserIDString(ctx))
}
