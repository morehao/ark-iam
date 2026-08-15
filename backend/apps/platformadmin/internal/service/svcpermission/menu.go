package svcpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func menuVisible(entity *model.MenuEntity) bool {
	return entity != nil && entity.ID != ""
}

type MenuSvc interface {
	Create(ctx *gin.Context, req *dtopermission.MenuCreateReq) (*dtopermission.MenuCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.MenuDeleteReq) error
	Update(ctx *gin.Context, req *dtopermission.MenuUpdateReq) error
	Detail(ctx *gin.Context, req *dtopermission.MenuDetailReq) (*dtopermission.MenuDetailResp, error)
	PageList(ctx *gin.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error)
	Tree(ctx *gin.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error)
}

type menuSvc struct{}

var _ MenuSvc = (*menuSvc)(nil)

func NewMenuSvc() MenuSvc {
	return &menuSvc{}
}

func (svc *menuSvc) Create(ctx *gin.Context, req *dtopermission.MenuCreateReq) (*dtopermission.MenuCreateResp, error) {
	insertEntity := &model.MenuEntity{
		AppID:        req.AppID,
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
		CreatedBy:    gctx.GetUserID(ctx),
	}

	if err := dao.NewMenuDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateMenu] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	return &dtopermission.MenuCreateResp{
		MenuID: insertEntity.ID,
	}, nil
}

func (svc *menuSvc) Delete(ctx *gin.Context, req *dtopermission.MenuDeleteReq) error {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	if !menuVisible(menuEntity) {
		return code.GetError(code.MenuNotExistError)
	}

	userID := gctx.GetUserID(ctx)
	if err := dao.NewMenuDao().Delete(ctx, req.MenuID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteMenu] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	return nil
}

func (svc *menuSvc) Update(ctx *gin.Context, req *dtopermission.MenuUpdateReq) error {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	if !menuVisible(menuEntity) {
		return code.GetError(code.MenuNotExistError)
	}

	userID := gctx.GetUserID(ctx)
	updateMap := map[string]any{
		"app_id":        req.AppID,
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
		glog.Errorf(ctx, "[svcpermission.UpdateMenu] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	return nil
}

func (svc *menuSvc) Detail(ctx *gin.Context, req *dtopermission.MenuDetailReq) (*dtopermission.MenuDetailResp, error) {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
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

func (svc *menuSvc) PageList(ctx *gin.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error) {
	menuRepo := dao.NewMenuDao()
	cond := &dao.MenuCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		AppID:    req.AppID,
		ParentID: req.ParentID,
		Name:     req.Name,
		Code:     req.Code,
		Type:     req.Type,
		Status:   req.Status,
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
	return &dtopermission.MenuPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *menuSvc) Tree(ctx *gin.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error) {
	menuRepo := dao.NewMenuDao()
	cond := &dao.MenuCond{
		AppID: req.AppID,
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

	return &dtopermission.MenuTreeResp{
		List: buildTree(""),
	}, nil
}
