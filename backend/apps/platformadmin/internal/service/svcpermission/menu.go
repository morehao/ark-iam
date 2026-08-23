package svcpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/pkg/iam/svcmenu"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func menuVisible(entity *model.MenuEntity) bool {
	return entity != nil && entity.ID != ""
}

// validateMenuEnums 校验菜单字典枚举合法值：Type/Status/Visibility 必须命中间断白名单常量。
func validateMenuEnums(req *objpermission.MenuBaseInfo) bool {
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
	return true
}

type MenuSvc interface {
	Create(ctx *gin.Context, req *dtopermission.MenuCreateReq) (*dtopermission.MenuCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.MenuDeleteReq) error
	Update(ctx *gin.Context, req *dtopermission.MenuUpdateReq) error
	Detail(ctx *gin.Context, req *dtopermission.MenuDetailReq) (*dtopermission.MenuDetailResp, error)
	PageList(ctx *gin.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error)
	Tree(ctx *gin.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error)
	MyTree(ctx *gin.Context) (*dtopermission.MenuMyTreeResp, error)
}

type menuSvc struct{}

var _ MenuSvc = (*menuSvc)(nil)

func NewMenuSvc() MenuSvc {
	return &menuSvc{}
}

func (svc *menuSvc) Create(ctx *gin.Context, req *dtopermission.MenuCreateReq) (*dtopermission.MenuCreateResp, error) {
	if !validateMenuEnums(&req.MenuBaseInfo) {
		return nil, code.GetError(code.MenuCreateError)
	}
	insertEntity := &model.MenuEntity{
		AppID:        req.AppID,
		ParentID:     req.ParentID,
		Name:         req.Name,
		Code:         req.Code,
		Path:         req.Path,
		Icon:         req.Icon,
		Sort:         req.Sort,
		Type:         req.Type,
		Visibility:   req.Visibility,
		Component:    req.Component,
		Redirect:     req.Redirect,
		Hidden:       req.Hidden,
		ExternalLink: req.ExternalLink,
		KeepAlive:    req.KeepAlive,
		Status:       req.Status,
		CreatedBy:    gincontext.GetUserIDString(ctx),
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

	userID := gincontext.GetUserIDString(ctx)
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
	if !validateMenuEnums(&req.MenuBaseInfo) {
		return code.GetError(code.MenuUpdateError)
	}

	userID := gincontext.GetUserIDString(ctx)
	updateMap := map[string]any{
		"app_id":        req.AppID,
		"parent_id":     req.ParentID,
		"name":          req.Name,
		"code":          req.Code,
		"path":          req.Path,
		"icon":          req.Icon,
		"sort":          req.Sort,
		"type":          req.Type,
		"visibility":    req.Visibility,
		"component":     req.Component,
		"redirect":      req.Redirect,
		"hidden":        req.Hidden,
		"external_link": req.ExternalLink,
		"keep_alive":    req.KeepAlive,
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

func (svc *menuSvc) PageList(ctx *gin.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error) {
	menuRepo := dao.NewMenuDao()
	cond := &dao.MenuCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
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
						Visibility:   menu.Visibility,
						Component:    menu.Component,
						Redirect:     menu.Redirect,
						Hidden:       menu.Hidden,
						ExternalLink: menu.ExternalLink,
						KeepAlive:    menu.KeepAlive,
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

// platformAdminAppCode 平台管理后台应用的种子编码（见 pkg/seed，appCodeAdmin）。
const platformAdminAppCode = "platform-admin"

// MyTree 返回当前用户可见的平台菜单树（侧边栏动态菜单）。
// 平台菜单固定归属平台管理后台应用（platform-admin），按用户的角色授权（超管全量 + role_menu 授权 + visibility）过滤，
// 与租户控制台菜单逻辑保持一致，复用公共层 svcmenu。
func (svc *menuSvc) MyTree(ctx *gin.Context) (*dtopermission.MenuMyTreeResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userID := gincontext.GetUserIDString(ctx)

	appList, _, err := dao.NewApplicationDao().GetPageListByCond(ctx, &dao.ApplicationCond{
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

	nodes, err := svcmenu.BuildMyMenuTree(ctx, tenantID, userID, []string{appList[0].ID})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.MyTree] build my menu tree fail, err:%v", err)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	return &dtopermission.MenuMyTreeResp{List: nodes}, nil
}
