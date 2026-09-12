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
	"github.com/morehao/golib/dbaccess/gormdao"
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

// Apps 当前租户订阅的启用应用（角色归属/菜单授权的应用选项，含内置应用如管理后台）。
func (svc *tenantMenuSvc) Apps(ctx *gin.Context) (*dtotenant.TenantAppsResp, error) {
	appList, err := loadSubscribedApps(ctx)
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

// loadSubscribedApps 当前租户订阅的启用应用（含内置应用，如管理后台）。
// 两道门槛都需满足：订阅关系 tenant_application.status=enable，且应用本身 application.status=enable
// —— 应用被停用后不应再把它的菜单/角色归属继续暴露给已订阅租户。
// 角色归属/应用名映射的应用选项集合：凡租户订阅且启用的应用均可选，
// 不再区分是否内置（`application.source=builtin` 只用于保护内置记录不被删除/篡改）。
func loadSubscribedApps(ctx *gin.Context) ([]model.ApplicationEntity, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	tenantAppList, _, err := dao.NewTenantApplicationDao().GetPageListByCond(ctx, &dao.TenantApplicationCond{
		TenantID: tenantID,
		Status:   model.AppStatusEnable,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.loadSubscribedApps] dao tenantApplication GetPageListByCond fail, err:%v, tenantID:%s", err, tenantID)
		return nil, code.GetError(code.MenuGetPageListError)
	}

	appIDSet := make(map[string]struct{}, len(tenantAppList))
	appIDs := make([]string, 0, len(tenantAppList))
	for _, item := range tenantAppList {
		if item.AppID == "" {
			continue
		}
		if _, ok := appIDSet[item.AppID]; ok {
			continue
		}
		appIDSet[item.AppID] = struct{}{}
		appIDs = append(appIDs, item.AppID)
	}
	if len(appIDs) == 0 {
		return nil, nil
	}
	// 下拉/映射顺序与平台侧应用排序一致（application.sort），避免多应用时顺序漂移
	appList, err := dao.NewApplicationDao().GetListByCond(ctx, &dao.ApplicationCond{
		BaseCond: &gormdao.BaseCond{OrderField: "sort, code"},
		IDs:      appIDs,
		Status:   model.AppStatusEnable,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.loadSubscribedApps] dao application GetListByCond fail, err:%v", err)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	return appList, nil
}

// tenantAdminAppCode 租户自服务应用的种子编码（见 pkg/seed，appCodeTenantAdmin）。
// 它是本控制台菜单的载体：内置应用的菜单默认不并入租户侧边栏，唯独它必须留下。
const tenantAdminAppCode = "tenant_admin"

// loadConsoleApps 租户控制台菜单范围的订阅应用：订阅且启用的应用。
// 内置应用（source=builtin）的菜单归属各自专属控制台，不并入租户控制台侧边栏——
// 例如管理后台（platform_admin）的「租户管理/应用管理」不能串台到租户侧边栏；
// 但租户自服务（tenant_admin）同样是内置应用、又是本控制台菜单的唯一载体，必须保留，否则侧边栏会空。
// 即：内置性只决定「是否并入本控制台」，与 loadSubscribedApps 的「角色可选应用」口径不同（后者含全部内置应用）。
func loadConsoleApps(ctx *gin.Context) ([]model.ApplicationEntity, error) {
	appList, err := loadSubscribedApps(ctx)
	if err != nil {
		return nil, err
	}
	consoleApps := make([]model.ApplicationEntity, 0, len(appList))
	for _, app := range appList {
		if app.Source.IsBuiltin() && app.Code != tenantAdminAppCode {
			continue
		}
		consoleApps = append(consoleApps, app)
	}
	return consoleApps, nil
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
	appList, err := loadConsoleApps(ctx)
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
//   - 内置管理员豁免：持有内置管理员角色（source=builtin && admin_type=admin）→ 全量菜单（含 visibility=admin，免授权）；
//   - 普通成员：按该用户授权菜单集合（role_menu 并集）过滤 + visibility 门槛（public/member）二次过滤；
//     父子收敛：父未达标/未授权时若存在可见子项则保留父壳，保证层级连贯。
//
// 应用范围取租户控制台应用（loadConsoleApps）：内置应用（source=builtin）的菜单由其专属控制台呈现，不并入本控制台。
func buildMyMenuTree(ctx *gin.Context) ([]dtotenant.MenuTreeItem, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userID := gincontext.GetUserIDString(ctx)
	appList, err := loadConsoleApps(ctx)
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

// userHoldsBuiltinAdmin 判断当前用户是否持有内置管理员角色（source=builtin && admin_type=admin）。
// 保留薄包装以复用公共层实现并维持既有测试契约。
func userHoldsBuiltinAdmin(ctx *gin.Context) (bool, error) {
	return svcmenu.UserHoldsBuiltinAdmin(ctx, gincontext.GetTenantIDString(ctx), gincontext.GetUserIDString(ctx))
}

// HasSystemAdminCapability 判断当前用户（按 gin 上下文取租户/用户）是否具备「系统管理能力」
// （admin_type == admin）：任一角色为管理员类型即具备。
func HasSystemAdminCapability(ctx *gin.Context) (bool, error) {
	adminType, err := ResolveUserAdminType(ctx)
	if err != nil {
		return false, err
	}
	return adminType.HasSystemAdmin(), nil
}

// requireSystemAdmin 校验当前操作者具备系统管理能力（admin_type=admin），否则返回能力不足错误。
// 租户自服务控制台定位为「管理层专用」：部门/用户/角色/密钥等管理写操作统一以此硬门槛兜底，
// 菜单可见性仅是 UX 层（前端隐藏不是安全边界），直接调用 API 也必须被拒。
// opErr 仅在系统错误（角色查询失败等）时兜底返回。
func requireSystemAdmin(ctx *gin.Context, opErr int) error {
	ok, err := HasSystemAdminCapability(ctx)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.requireSystemAdmin] resolve admin level fail, err:%v", err)
		return code.GetError(opErr)
	}
	if !ok {
		return code.GetError(code.UserSystemAdminRequiredError)
	}
	return nil
}

// ResolveUserAdminType 推导当前用户的系统管理类型：聚合该用户全部角色，
// 任一角色为管理员类型（admin）即视为管理员类型，否则为普通类型（normal）。复用公共层 svcmenu。
func ResolveUserAdminType(ctx *gin.Context) (model.SysAdminType, error) {
	return svcmenu.ResolveUserAdminType(ctx, gincontext.GetTenantIDString(ctx), gincontext.GetUserIDString(ctx))
}
