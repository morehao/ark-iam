package svctenant

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

// tenantMenuRepository 租户侧菜单查询依赖
type tenantMenuRepository interface {
	GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.MenuEntityList, int64, error)
}

var newTenantMenuRepo = func() tenantMenuRepository {
	return dao.NewMenuDao()
}

// TenantMenuSvc 租户侧菜单服务
type TenantMenuSvc interface {
	Tree(ctx *gin.Context) (*dtotenant.MenuTreeResp, error)
}

type tenantMenuSvc struct{}

var _ TenantMenuSvc = (*tenantMenuSvc)(nil)

func NewTenantMenuSvc() TenantMenuSvc {
	return &tenantMenuSvc{}
}

func (svc *tenantMenuSvc) Tree(ctx *gin.Context) (*dtotenant.MenuTreeResp, error) {
	tenantID := gincontext.GetTenantID(ctx)

	// 当前租户订阅的启用应用
	tenantAppList, _, err := dao.NewTenantApplicationDao().GetPageListByCond(ctx, &dao.TenantApplicationCond{
		TenantID: tenantID,
		Status:   model.AppStatusEnable,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantMenuTree] dao tenantApplication GetPageListByCond fail, err:%v, tenantID:%d", err, tenantID)
		return nil, code.GetError(code.MenuGetPageListError)
	}

	// 收集订阅应用，排除平台系统内置应用（如管理后台），租户自服务只展示租户可用的应用菜单
	appIDs := make(map[uint]struct{})
	for _, item := range tenantAppList {
		if item.AppID == 0 {
			continue
		}
		appEntity, err := dao.NewApplicationDao().GetByID(ctx, item.AppID)
		if err != nil || appEntity == nil || appEntity.ID == 0 {
			glog.Warnf(ctx, "[svctenant.TenantMenuTree] application GetByID fail or not exist, err:%v, appID:%d", err, item.AppID)
			continue
		}
		if appEntity.IsSystem == 1 {
			continue
		}
		appIDs[item.AppID] = struct{}{}
	}

	menuRepo := newTenantMenuRepo()
	var menus []*model.MenuEntity
	for appID := range appIDs {
		menuEntityList, _, err := menuRepo.GetPageListByCond(ctx, &dao.MenuCond{
			AppID:  appID,
			Status: model.AppStatusEnable,
		})
		if err != nil {
			glog.Errorf(ctx, "[svctenant.TenantMenuTree] dao menu GetPageListByCond fail, err:%v, appID:%d", err, appID)
			return nil, code.GetError(code.MenuGetPageListError)
		}
		for i := range menuEntityList {
			menus = append(menus, &menuEntityList[i])
		}
	}

	var buildTree func(parentID uint) []dtotenant.MenuTreeItem
	buildTree = func(parentID uint) []dtotenant.MenuTreeItem {
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

	return &dtotenant.MenuTreeResp{
		List: buildTree(0),
	}, nil
}
