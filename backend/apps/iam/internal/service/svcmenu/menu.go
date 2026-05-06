package svcmenu

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtomenu"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objmenu"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type MenuSvc interface {
	Create(ctx *gin.Context, req *dtomenu.MenuCreateReq) (*dtomenu.MenuCreateResp, error)
	Delete(ctx *gin.Context, req *dtomenu.MenuDeleteReq) error
	Update(ctx *gin.Context, req *dtomenu.MenuUpdateReq) error
	Detail(ctx *gin.Context, req *dtomenu.MenuDetailReq) (*dtomenu.MenuDetailResp, error)
	PageList(ctx *gin.Context, req *dtomenu.MenuPageListReq) (*dtomenu.MenuPageListResp, error)
	Tree(ctx *gin.Context, req *dtomenu.MenuTreeReq) (*dtomenu.MenuTreeResp, error)
}

type menuSvc struct {
}

var _ MenuSvc = (*menuSvc)(nil)

func NewMenuSvc() MenuSvc {
	return &menuSvc{}
}

func (svc *menuSvc) Create(ctx *gin.Context, req *dtomenu.MenuCreateReq) (*dtomenu.MenuCreateResp, error) {
	insertEntity := &model.MenuEntity{
		TenantID:     req.TenantID,
		ParentID:     req.ParentID,
		Name:         req.Name,
		Code:         req.Code,
		Path:         req.Path,
		Icon:         req.Icon,
		Sort:         req.Sort,
		Type:         req.Type,
		Component:    req.Component,
		Redirect:     req.Redirect,
		Hidden:       req.Hidden,
		ExternalLink: req.ExternalLink,
		KeepAlive:    req.KeepAlive,
		Permission:   req.Permission,
		Status:       req.Status,
		CreatedBy:    gincontext.GetUserID(ctx),
	}

	if err := dao.NewMenuDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcmenu.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	return &dtomenu.MenuCreateResp{
		MenuID: insertEntity.ID,
	}, nil
}

func (svc *menuSvc) Delete(ctx *gin.Context, req *dtomenu.MenuDeleteReq) error {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcmenu.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	if menuEntity == nil || menuEntity.ID == 0 {
		return code.GetError(code.MenuNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewMenuDao().Delete(ctx, req.MenuID, userID); err != nil {
		glog.Errorf(ctx, "[svcmenu.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	return nil
}

func (svc *menuSvc) Update(ctx *gin.Context, req *dtomenu.MenuUpdateReq) error {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcmenu.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	if menuEntity == nil || menuEntity.ID == 0 {
		return code.GetError(code.MenuNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":     req.TenantID,
		"parent_id":     req.ParentID,
		"name":          req.Name,
		"code":          req.Code,
		"path":          req.Path,
		"icon":          req.Icon,
		"sort":          req.Sort,
		"type":          req.Type,
		"component":     req.Component,
		"redirect":      req.Redirect,
		"hidden":        req.Hidden,
		"external_link": req.ExternalLink,
		"keep_alive":    req.KeepAlive,
		"permission":    req.Permission,
		"status":        req.Status,
		"updated_by":    userID,
	}
	if err := dao.NewMenuDao().UpdateMap(ctx, req.MenuID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcmenu.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	return nil
}

func (svc *menuSvc) Detail(ctx *gin.Context, req *dtomenu.MenuDetailReq) (*dtomenu.MenuDetailResp, error) {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcmenu.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetDetailError)
	}
	if menuEntity == nil || menuEntity.ID == 0 {
		return nil, code.GetError(code.MenuNotExistError)
	}

	resp := &dtomenu.MenuDetailResp{
		MenuID: menuEntity.ID,
		MenuBaseInfo: objmenu.MenuBaseInfo{
			TenantID:     menuEntity.TenantID,
			ParentID:     menuEntity.ParentID,
			Name:         menuEntity.Name,
			Code:         menuEntity.Code,
			Path:         menuEntity.Path,
			Icon:         menuEntity.Icon,
			Sort:         menuEntity.Sort,
			Type:         menuEntity.Type,
			Component:    menuEntity.Component,
			Redirect:     menuEntity.Redirect,
			Hidden:       menuEntity.Hidden,
			ExternalLink: menuEntity.ExternalLink,
			KeepAlive:    menuEntity.KeepAlive,
			Permission:   menuEntity.Permission,
			Status:       menuEntity.Status,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: menuEntity.CreatedAt.Unix(),
			UpdatedAt: menuEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *menuSvc) PageList(ctx *gin.Context, req *dtomenu.MenuPageListReq) (*dtomenu.MenuPageListResp, error) {
	cond := &dao.MenuCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		ParentID: req.ParentID,
		Name:     req.Name,
		Code:     req.Code,
		Type:     req.Type,
		Status:   req.Status,
	}
	menuEntityList, total, err := dao.NewMenuDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcmenu.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetPageListError)
	}

	list := make([]dtomenu.MenuPageListItem, 0, len(menuEntityList))
	for _, v := range menuEntityList {
		list = append(list, dtomenu.MenuPageListItem{
			MenuID: v.ID,
			MenuBaseInfo: objmenu.MenuBaseInfo{
				TenantID:     v.TenantID,
				ParentID:     v.ParentID,
				Name:         v.Name,
				Code:         v.Code,
				Path:         v.Path,
				Icon:         v.Icon,
				Sort:         v.Sort,
				Type:         v.Type,
				Component:    v.Component,
				Redirect:     v.Redirect,
				Hidden:       v.Hidden,
				ExternalLink: v.ExternalLink,
				KeepAlive:    v.KeepAlive,
				Permission:   v.Permission,
				Status:       v.Status,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtomenu.MenuPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *menuSvc) Tree(ctx *gin.Context, req *dtomenu.MenuTreeReq) (*dtomenu.MenuTreeResp, error) {
	cond := &dao.MenuCond{
		TenantID: req.TenantID,
	}
	menuEntityList, _, err := dao.NewMenuDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcmenu.Tree] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetPageListError)
	}

	var buildTree func(parentID uint) []dtomenu.MenuTreeItem
	buildTree = func(parentID uint) []dtomenu.MenuTreeItem {
		var items []dtomenu.MenuTreeItem
		for _, menu := range menuEntityList {
			if menu.ParentID == parentID {
				item := dtomenu.MenuTreeItem{
					MenuID: menu.ID,
					MenuBaseInfo: objmenu.MenuBaseInfo{
						TenantID:     menu.TenantID,
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
					OperatorBaseInfo: gobject.OperatorBaseInfo{
						UpdatedAt: menu.UpdatedAt.Unix(),
					},
					Children: buildTree(menu.ID),
				}
				items = append(items, item)
			}
		}
		return items
	}

	return &dtomenu.MenuTreeResp{
		List: buildTree(0),
	}, nil
}