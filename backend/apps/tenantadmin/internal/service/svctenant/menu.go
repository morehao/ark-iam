package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
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
	tree, err := buildTenantMenuTree(ctx)
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
	tenantID := gctx.GetTenantID(ctx)
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
