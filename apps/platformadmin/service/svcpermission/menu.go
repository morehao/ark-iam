package svcpermission

import (
	"context"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtopermission"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/menu"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func menuVisible(entity *model.MenuEntity) bool {
	return entity != nil && entity.ID != ""
}

type MenuSvc interface {
	Detail(ctx context.Context, req *dtopermission.MenuDetailReq) (*dtopermission.MenuDetailResp, error)
	PageList(ctx context.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error)
	Tree(ctx context.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error)
	MyTree(ctx context.Context, req *dtopermission.MenuMyTreeReq) (*dtopermission.MenuMyTreeResp, error)
}

type menuSvc struct {
	db gormdao.DBGetter
}

var _ MenuSvc = (*menuSvc)(nil)

func NewMenuSvc(db gormdao.DBGetter) MenuSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &menuSvc{db: db}
}

func (svc *menuSvc) Detail(ctx context.Context, req *dtopermission.MenuDetailReq) (*dtopermission.MenuDetailResp, error) {
	menuEntity, err := dao.NewMenuDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetDetailError)
	}
	if !menuVisible(menuEntity) {
		return nil, code.GetError(code.MenuNotExistError)
	}

	resp := &dtopermission.MenuDetailResp{
		MenuID: menuEntity.ID,
		MenuBaseInfo: objpermission.MenuBaseInfo{
			AppID:        menuEntity.AppID,
			ParentID:     menuEntity.ParentID,
			Name:         menuEntity.Name,
			Code:         menuEntity.Code,
			Path:         menuEntity.Path,
			Icon:         menuEntity.Icon,
			Sort:         menuEntity.Sort,
			Type:         menuEntity.Type,
			Visibility:   menuEntity.Visibility,
			Component:    menuEntity.Component,
			Redirect:     menuEntity.Redirect,
			Hidden:       menuEntity.Hidden,
			ExternalLink: menuEntity.ExternalLink,
			KeepAlive:    menuEntity.KeepAlive,
			Status:       menuEntity.Status,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: menuEntity.CreatedAt.Unix(),
			UpdatedAt: menuEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *menuSvc) PageList(ctx context.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error) {
	menuRepo := dao.NewMenuDao(dao.WithDBGetter(svc.db))
	cond := &dao.MenuCond{
		BaseCond: &gormdao.BaseCond{
			Page:       req.Page,
			PageSize:   req.PageSize,
			OrderField: dao.MenuOrderBySort,
		},
		AppID:      req.AppID,
		ParentID:   req.ParentID,
		Name:       req.Name,
		Code:       req.Code,
		Type:       req.Type,
		Status:     req.Status,
		Visibility: req.Visibility,
	}
	menuEntityList, total, err := menuRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListMenu] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetPageListError)
	}

	list := make([]dtopermission.MenuPageListItem, 0, len(menuEntityList))
	for _, v := range menuEntityList {
		list = append(list, dtopermission.MenuPageListItem{
			MenuID: v.ID,
			MenuBaseInfo: objpermission.MenuBaseInfo{
				AppID:        v.AppID,
				ParentID:     v.ParentID,
				Name:         v.Name,
				Code:         v.Code,
				Path:         v.Path,
				Icon:         v.Icon,
				Sort:         v.Sort,
				Type:         v.Type,
				Visibility:   v.Visibility,
				Component:    v.Component,
				Redirect:     v.Redirect,
				Hidden:       v.Hidden,
				ExternalLink: v.ExternalLink,
				KeepAlive:    v.KeepAlive,
				Status:       v.Status,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				CreatedAt: v.CreatedAt.Unix(),
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtopermission.MenuPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *menuSvc) Tree(ctx context.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error) {
	menuRepo := dao.NewMenuDao(dao.WithDBGetter(svc.db))
	// 菜单管理页展示顺序与两端侧边栏同源（dao.MenuOrderBySort）：控制台里改「排序」后，
	// 这里与侧边栏必须同时按新顺序呈现，否则运维无法确认排序是否生效。
	cond := &dao.MenuCond{
		BaseCond: &gormdao.BaseCond{OrderField: dao.MenuOrderBySort},
		AppID:    req.AppID,
	}
	menuEntityList, _, err := menuRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.TreeMenu] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetPageListError)
	}

	var buildTree func(parentID string) []dtopermission.MenuTreeItem
	buildTree = func(parentID string) []dtopermission.MenuTreeItem {
		var items []dtopermission.MenuTreeItem
		for _, menu := range menuEntityList {
			if menu.ParentID == parentID {
				item := dtopermission.MenuTreeItem{
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
					OperatorBaseInfo: gobject.OperatorBaseInfo{
						CreatedAt: menu.CreatedAt.Unix(),
						UpdatedAt: menu.UpdatedAt.Unix(),
					},
					Children: buildTree(menu.ID),
				}
				items = append(items, item)
			}
		}
		return items
	}

	return &dtopermission.MenuTreeResp{
		List: buildTree(""),
	}, nil
}

// platformAdminAppCode 平台管理后台应用的种子编码（见 pkg/seed，appCodeAdmin）。
const platformAdminAppCode = "platform_admin"

// MyTree 返回当前用户可见的平台菜单树（侧边栏动态菜单）。
// 平台菜单固定归属平台管理后台应用（platform_admin），按用户的角色授权（管理员角色全量 + role_menu 授权 + visibility）过滤，
// 与租户控制台菜单逻辑保持一致，复用公共层 menu。
func (svc *menuSvc) MyTree(ctx context.Context, req *dtopermission.MenuMyTreeReq) (*dtopermission.MenuMyTreeResp, error) {
	tenantID := dbclient.TenantIDFromContext(ctx)
	userID := req.Operator.UserID

	appList, _, err := dao.NewApplicationDao(dao.WithDBGetter(svc.db)).GetPageListByCond(ctx, &dao.ApplicationCond{
		BaseCond: &gormdao.BaseCond{Page: 1, PageSize: 1},
		Code:     platformAdminAppCode,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.MyTree] query platform app fail, err:%v", err)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	if len(appList) == 0 || appList[0].ID == "" {
		glog.Errorf(ctx, "[svcpermission.MyTree] platform app not found, code:%s", platformAdminAppCode)
		return nil, code.GetError(code.MenuGetPageListError)
	}

	nodes, err := menu.BuildMyMenuTree(ctx, tenantID, userID, []string{appList[0].ID})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.MyTree] build my menu tree fail, err:%v", err)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	return &dtopermission.MenuMyTreeResp{List: nodes}, nil
}
