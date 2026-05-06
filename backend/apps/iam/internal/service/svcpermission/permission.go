package svcpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objpermission"
	"github.com/morehao/ark-iam/iam/object/objresource"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type PermissionSvc interface {
	CreateRole(ctx *gin.Context, req *dtopermission.RoleCreateReq) (*dtopermission.RoleCreateResp, error)
	DeleteRole(ctx *gin.Context, req *dtopermission.RoleDeleteReq) error
	UpdateRole(ctx *gin.Context, req *dtopermission.RoleUpdateReq) error
	DetailRole(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error)
	PageListRole(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error)
	CreateMenu(ctx *gin.Context, req *dtopermission.MenuCreateReq) (*dtopermission.MenuCreateResp, error)
	DeleteMenu(ctx *gin.Context, req *dtopermission.MenuDeleteReq) error
	UpdateMenu(ctx *gin.Context, req *dtopermission.MenuUpdateReq) error
	DetailMenu(ctx *gin.Context, req *dtopermission.MenuDetailReq) (*dtopermission.MenuDetailResp, error)
	PageListMenu(ctx *gin.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error)
	TreeMenu(ctx *gin.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error)
	CreateResource(ctx *gin.Context, req *dtopermission.ResourceCreateReq) (*dtopermission.ResourceCreateResp, error)
	DeleteResource(ctx *gin.Context, req *dtopermission.ResourceDeleteReq) error
	UpdateResource(ctx *gin.Context, req *dtopermission.ResourceUpdateReq) error
	DetailResource(ctx *gin.Context, req *dtopermission.ResourceDetailReq) (*dtopermission.ResourceDetailResp, error)
	PageListResource(ctx *gin.Context, req *dtopermission.ResourcePageListReq) (*dtopermission.ResourcePageListResp, error)
	CreateScope(ctx *gin.Context, req *dtopermission.ScopeCreateReq) (*dtopermission.ScopeCreateResp, error)
	DeleteScope(ctx *gin.Context, req *dtopermission.ScopeDeleteReq) error
	UpdateScope(ctx *gin.Context, req *dtopermission.ScopeUpdateReq) error
	DetailScope(ctx *gin.Context, req *dtopermission.ScopeDetailReq) (*dtopermission.ScopeDetailResp, error)
	PageListScope(ctx *gin.Context, req *dtopermission.ScopePageListReq) (*dtopermission.ScopePageListResp, error)
	CreateRoleMenu(ctx *gin.Context, req *dtopermission.RoleMenuCreateReq) (*dtopermission.RoleMenuCreateResp, error)
	DeleteRoleMenu(ctx *gin.Context, req *dtopermission.RoleMenuDeleteReq) error
	PageListRoleMenu(ctx *gin.Context, req *dtopermission.RoleMenuPageListReq) (*dtopermission.RoleMenuPageListResp, error)
	CreateRoleScope(ctx *gin.Context, req *dtopermission.RoleScopeCreateReq) (*dtopermission.RoleScopeCreateResp, error)
	DeleteRoleScope(ctx *gin.Context, req *dtopermission.RoleScopeDeleteReq) error
	PageListRoleScope(ctx *gin.Context, req *dtopermission.RoleScopePageListReq) (*dtopermission.RoleScopePageListResp, error)
	CreateUserRole(ctx *gin.Context, req *dtopermission.UserRoleCreateReq) (*dtopermission.UserRoleCreateResp, error)
	DeleteUserRole(ctx *gin.Context, req *dtopermission.UserRoleDeleteReq) error
	PageListUserRole(ctx *gin.Context, req *dtopermission.UserRolePageListReq) (*dtopermission.UserRolePageListResp, error)
}

type permissionSvc struct {
}

var _ PermissionSvc = (*permissionSvc)(nil)

func NewPermissionSvc() PermissionSvc {
	return &permissionSvc{}
}

func (svc *permissionSvc) CreateRole(ctx *gin.Context, req *dtopermission.RoleCreateReq) (*dtopermission.RoleCreateResp, error) {
	insertEntity := &model.RoleEntity{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Type:        req.Type,
		IsDefault:   req.IsDefault,
		CreatedBy:   gincontext.GetUserID(ctx),
	}

	if err := dao.NewRoleDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRole] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleCreateError)
	}
	return &dtopermission.RoleCreateResp{
		RoleID: insertEntity.ID,
	}, nil
}

func (svc *permissionSvc) DeleteRole(ctx *gin.Context, req *dtopermission.RoleDeleteReq) error {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleDeleteError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return code.GetError(code.RoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewRoleDao().Delete(ctx, req.RoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRole] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleDeleteError)
	}
	return nil
}

func (svc *permissionSvc) UpdateRole(ctx *gin.Context, req *dtopermission.RoleUpdateReq) error {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return code.GetError(code.RoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":    req.TenantID,
		"name":         req.Name,
		"code":         req.Code,
		"description":  req.Description,
		"type":         req.Type,
		"is_default":   req.IsDefault,
		"updated_by":   userID,
	}
	if err := dao.NewRoleDao().UpdateMap(ctx, req.RoleID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateRole] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	return nil
}

func (svc *permissionSvc) DetailRole(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error) {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return nil, code.GetError(code.RoleNotExistError)
	}

	resp := &dtopermission.RoleDetailResp{
		RoleID: roleEntity.ID,
		RoleBaseInfo: objpermission.RoleBaseInfo{
			TenantID:    roleEntity.TenantID,
			Name:        roleEntity.Name,
			Code:        roleEntity.Code,
			Description: roleEntity.Description,
			Type:        roleEntity.Type,
			IsDefault:   roleEntity.IsDefault,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: int64(roleEntity.CreatedAt.Unix()),
			UpdatedAt: int64(roleEntity.UpdatedAt.Unix()),
		},
	}
	return resp, nil
}

func (svc *permissionSvc) PageListRole(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error) {
	cond := &dao.RoleCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		Name:     req.Name,
		Code:     req.Code,
		Type:     req.Type,
	}
	roleEntityList, total, err := dao.NewRoleDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListRole] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetPageListError)
	}

	list := make([]dtopermission.RolePageListItem, 0, len(roleEntityList))
	for _, v := range roleEntityList {
		list = append(list, dtopermission.RolePageListItem{
			RoleID: v.ID,
			RoleBaseInfo: objpermission.RoleBaseInfo{
				TenantID:    v.TenantID,
				Name:        v.Name,
				Code:        v.Code,
				Description: v.Description,
				Type:        v.Type,
				IsDefault:   v.IsDefault,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: int64(v.UpdatedAt.Unix()),
			},
		})
	}
	return &dtopermission.RolePageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *permissionSvc) CreateMenu(ctx *gin.Context, req *dtopermission.MenuCreateReq) (*dtopermission.MenuCreateResp, error) {
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
		glog.Errorf(ctx, "[svcpermission.CreateMenu] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	return &dtopermission.MenuCreateResp{
		MenuID: insertEntity.ID,
	}, nil
}

func (svc *permissionSvc) DeleteMenu(ctx *gin.Context, req *dtopermission.MenuDeleteReq) error {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	if menuEntity == nil || menuEntity.ID == 0 {
		return code.GetError(code.MenuNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewMenuDao().Delete(ctx, req.MenuID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteMenu] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	return nil
}

func (svc *permissionSvc) UpdateMenu(ctx *gin.Context, req *dtopermission.MenuUpdateReq) error {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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
		glog.Errorf(ctx, "[svcpermission.UpdateMenu] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	return nil
}

func (svc *permissionSvc) DetailMenu(ctx *gin.Context, req *dtopermission.MenuDetailReq) (*dtopermission.MenuDetailResp, error) {
	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetDetailError)
	}
	if menuEntity == nil || menuEntity.ID == 0 {
		return nil, code.GetError(code.MenuNotExistError)
	}

	resp := &dtopermission.MenuDetailResp{
		MenuID: menuEntity.ID,
		MenuBaseInfo: objpermission.MenuBaseInfo{
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

func (svc *permissionSvc) PageListMenu(ctx *gin.Context, req *dtopermission.MenuPageListReq) (*dtopermission.MenuPageListResp, error) {
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
		glog.Errorf(ctx, "[svcpermission.PageListMenu] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetPageListError)
	}

	list := make([]dtopermission.MenuPageListItem, 0, len(menuEntityList))
	for _, v := range menuEntityList {
		list = append(list, dtopermission.MenuPageListItem{
			MenuID: v.ID,
			MenuBaseInfo: objpermission.MenuBaseInfo{
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
	return &dtopermission.MenuPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *permissionSvc) TreeMenu(ctx *gin.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error) {
	cond := &dao.MenuCond{
		TenantID: req.TenantID,
	}
	menuEntityList, _, err := dao.NewMenuDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.TreeMenu] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetPageListError)
	}

	var buildTree func(parentID uint) []dtopermission.MenuTreeItem
	buildTree = func(parentID uint) []dtopermission.MenuTreeItem {
		var items []dtopermission.MenuTreeItem
		for _, menu := range menuEntityList {
			if menu.ParentID == parentID {
				item := dtopermission.MenuTreeItem{
					MenuID: menu.ID,
					MenuBaseInfo: objpermission.MenuBaseInfo{
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

	return &dtopermission.MenuTreeResp{
		List: buildTree(0),
	}, nil
}

func (svc *permissionSvc) CreateResource(ctx *gin.Context, req *dtopermission.ResourceCreateReq) (*dtopermission.ResourceCreateResp, error) {
	insertEntity := &model.ResourceEntity{
		TenantID:       req.TenantID,
		Name:           req.Name,
		Indicator:      req.Indicator,
		IsDefault:      req.IsDefault,
		AccessTokenTtl: req.AccessTokenTtl,
		CreatedBy:      gincontext.GetUserID(ctx),
	}

	if err := dao.NewResourceDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateResource] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceCreateError)
	}
	return &dtopermission.ResourceCreateResp{
		ResourceID: insertEntity.ID,
	}, nil
}

func (svc *permissionSvc) DeleteResource(ctx *gin.Context, req *dtopermission.ResourceDeleteReq) error {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteResource] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceDeleteError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return code.GetError(code.ResourceNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewResourceDao().Delete(ctx, req.ResourceID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteResource] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceDeleteError)
	}
	return nil
}

func (svc *permissionSvc) UpdateResource(ctx *gin.Context, req *dtopermission.ResourceUpdateReq) error {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateResource] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceUpdateError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return code.GetError(code.ResourceNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":        req.TenantID,
		"name":            req.Name,
		"indicator":       req.Indicator,
		"is_default":      req.IsDefault,
		"access_token_ttl": req.AccessTokenTtl,
		"updated_by":      userID,
	}
	if err := dao.NewResourceDao().UpdateMap(ctx, req.ResourceID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateResource] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceUpdateError)
	}
	return nil
}

func (svc *permissionSvc) DetailResource(ctx *gin.Context, req *dtopermission.ResourceDetailReq) (*dtopermission.ResourceDetailResp, error) {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailResource] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceGetDetailError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return nil, code.GetError(code.ResourceNotExistError)
	}

	resp := &dtopermission.ResourceDetailResp{
		ResourceID: resourceEntity.ID,
		ResourceBaseInfo: objresource.ResourceBaseInfo{
			TenantID:       resourceEntity.TenantID,
			Name:           resourceEntity.Name,
			Indicator:      resourceEntity.Indicator,
			IsDefault:      resourceEntity.IsDefault,
			AccessTokenTtl: resourceEntity.AccessTokenTtl,
		},
	}
	return resp, nil
}

func (svc *permissionSvc) PageListResource(ctx *gin.Context, req *dtopermission.ResourcePageListReq) (*dtopermission.ResourcePageListResp, error) {
	cond := &dao.ResourceCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:  req.TenantID,
		Name:      req.Name,
		Indicator: req.Indicator,
	}
	resourceEntityList, total, err := dao.NewResourceDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListResource] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceGetPageListError)
	}

	list := make([]dtopermission.ResourcePageListItem, 0, len(resourceEntityList))
	for _, v := range resourceEntityList {
		list = append(list, dtopermission.ResourcePageListItem{
			ResourceID: v.ID,
			ResourceBaseInfo: objresource.ResourceBaseInfo{
				TenantID:       v.TenantID,
				Name:           v.Name,
				Indicator:      v.Indicator,
				IsDefault:      v.IsDefault,
				AccessTokenTtl: v.AccessTokenTtl,
			},
		})
	}
	return &dtopermission.ResourcePageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *permissionSvc) CreateScope(ctx *gin.Context, req *dtopermission.ScopeCreateReq) (*dtopermission.ScopeCreateResp, error) {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceGetDetailError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return nil, code.GetError(code.ResourceNotExistError)
	}

	insertEntity := &model.ScopeEntity{
		TenantID:   req.TenantID,
		ResourceID: req.ResourceID,
		Name:       req.Name,
		Description: req.Description,
		CreatedBy:  gincontext.GetUserID(ctx),
	}

	if err := dao.NewScopeDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateScope] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeCreateError)
	}
	return &dtopermission.ScopeCreateResp{
		ScopeID: insertEntity.ID,
	}, nil
}

func (svc *permissionSvc) DeleteScope(ctx *gin.Context, req *dtopermission.ScopeDeleteReq) error {
	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeDeleteError)
	}
	if scopeEntity == nil || scopeEntity.ID == 0 {
		return code.GetError(code.ScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewScopeDao().Delete(ctx, req.ScopeID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteScope] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeDeleteError)
	}
	return nil
}

func (svc *permissionSvc) UpdateScope(ctx *gin.Context, req *dtopermission.ScopeUpdateReq) error {
	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeUpdateError)
	}
	if scopeEntity == nil || scopeEntity.ID == 0 {
		return code.GetError(code.ScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":    req.TenantID,
		"resource_id":  req.ResourceID,
		"name":         req.Name,
		"description":  req.Description,
		"updated_by":   userID,
	}
	if err := dao.NewScopeDao().UpdateMap(ctx, req.ScopeID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateScope] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeUpdateError)
	}
	return nil
}

func (svc *permissionSvc) DetailScope(ctx *gin.Context, req *dtopermission.ScopeDetailReq) (*dtopermission.ScopeDetailResp, error) {
	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeGetDetailError)
	}
	if scopeEntity == nil || scopeEntity.ID == 0 {
		return nil, code.GetError(code.ScopeNotExistError)
	}

	resp := &dtopermission.ScopeDetailResp{
		ScopeID: scopeEntity.ID,
		ScopeBaseInfo: objresource.ScopeBaseInfo{
			TenantID:   scopeEntity.TenantID,
			ResourceID: scopeEntity.ResourceID,
			Name:       scopeEntity.Name,
			Description: scopeEntity.Description,
		},
	}
	return resp, nil
}

func (svc *permissionSvc) PageListScope(ctx *gin.Context, req *dtopermission.ScopePageListReq) (*dtopermission.ScopePageListResp, error) {
	cond := &dao.ScopeCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:   req.TenantID,
		ResourceID: req.ResourceID,
		Name:       req.Name,
	}
	scopeEntityList, total, err := dao.NewScopeDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListScope] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeGetPageListError)
	}

	list := make([]dtopermission.ScopePageListItem, 0, len(scopeEntityList))
	for _, v := range scopeEntityList {
		list = append(list, dtopermission.ScopePageListItem{
			ScopeID: v.ID,
			ScopeBaseInfo: objresource.ScopeBaseInfo{
				TenantID:   v.TenantID,
				ResourceID: v.ResourceID,
				Name:       v.Name,
				Description: v.Description,
			},
		})
	}
	return &dtopermission.ScopePageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *permissionSvc) CreateRoleMenu(ctx *gin.Context, req *dtopermission.RoleMenuCreateReq) (*dtopermission.RoleMenuCreateResp, error) {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRoleMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return nil, code.GetError(code.RoleNotExistError)
	}

	menuEntity, err := dao.NewMenuDao().GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRoleMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetDetailError)
	}
	if menuEntity == nil || menuEntity.ID == 0 {
		return nil, code.GetError(code.MenuNotExistError)
	}

	insertEntity := &model.RoleMenuEntity{
		TenantID: req.TenantID,
		RoleID:   req.RoleID,
		MenuID:   req.MenuID,
		CreatedBy: gincontext.GetUserID(ctx),
	}

	if err := dao.NewRoleMenuDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRoleMenu] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleMenuCreateError)
	}
	return &dtopermission.RoleMenuCreateResp{}, nil
}

func (svc *permissionSvc) DeleteRoleMenu(ctx *gin.Context, req *dtopermission.RoleMenuDeleteReq) error {
	roleMenuEntity, err := dao.NewRoleMenuDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleMenu] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleMenuDeleteError)
	}
	if roleMenuEntity == nil || roleMenuEntity.ID == 0 {
		return code.GetError(code.RoleMenuNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewRoleMenuDao().Delete(ctx, req.RoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleMenu] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleMenuDeleteError)
	}
	return nil
}

func (svc *permissionSvc) PageListRoleMenu(ctx *gin.Context, req *dtopermission.RoleMenuPageListReq) (*dtopermission.RoleMenuPageListResp, error) {
	cond := &dao.RoleMenuCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		RoleID:   req.RoleID,
		MenuID:   req.MenuID,
	}
	roleMenuEntityList, total, err := dao.NewRoleMenuDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListRoleMenu] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleMenuGetPageListError)
	}

	list := make([]dtopermission.RoleMenuPageListItem, 0, len(roleMenuEntityList))
	for _, v := range roleMenuEntityList {
		list = append(list, dtopermission.RoleMenuPageListItem{
			RoleID:   v.RoleID,
			MenuID:   v.MenuID,
			TenantID: v.TenantID,
		})
	}
	return &dtopermission.RoleMenuPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *permissionSvc) CreateRoleScope(ctx *gin.Context, req *dtopermission.RoleScopeCreateReq) (*dtopermission.RoleScopeCreateResp, error) {
	roleDao := dao.NewRoleDao()
	roleEntity, err := roleDao.GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRoleScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return nil, code.GetError(code.RoleNotExistError)
	}

	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRoleScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeGetDetailError)
	}
	if scopeEntity == nil || scopeEntity.ID == 0 {
		return nil, code.GetError(code.ScopeNotExistError)
	}

	insertEntity := &model.RoleScopeEntity{
		TenantID: req.TenantID,
		RoleID:   req.RoleID,
		ScopeID:  req.ScopeID,
		CreatedBy: gincontext.GetUserID(ctx),
	}

	if err := dao.NewRoleScopeDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRoleScope] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleScopeCreateError)
	}
	return &dtopermission.RoleScopeCreateResp{}, nil
}

func (svc *permissionSvc) DeleteRoleScope(ctx *gin.Context, req *dtopermission.RoleScopeDeleteReq) error {
	roleScopeEntity, err := dao.NewRoleScopeDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleScopeDeleteError)
	}
	if roleScopeEntity == nil || roleScopeEntity.ID == 0 {
		return code.GetError(code.RoleScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewRoleScopeDao().Delete(ctx, req.RoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleScope] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleScopeDeleteError)
	}
	return nil
}

func (svc *permissionSvc) PageListRoleScope(ctx *gin.Context, req *dtopermission.RoleScopePageListReq) (*dtopermission.RoleScopePageListResp, error) {
	cond := &dao.RoleScopeCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		RoleID:   req.RoleID,
		ScopeID:  req.ScopeID,
	}
	roleScopeEntityList, total, err := dao.NewRoleScopeDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListRoleScope] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleScopeGetPageListError)
	}

	list := make([]dtopermission.RoleScopePageListItem, 0, len(roleScopeEntityList))
	for _, v := range roleScopeEntityList {
		list = append(list, dtopermission.RoleScopePageListItem{
			RoleID:   v.RoleID,
			ScopeID:  v.ScopeID,
			TenantID: v.TenantID,
		})
	}
	return &dtopermission.RoleScopePageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *permissionSvc) CreateUserRole(ctx *gin.Context, req *dtopermission.UserRoleCreateReq) (*dtopermission.UserRoleCreateResp, error) {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateUserRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return nil, code.GetError(code.RoleNotExistError)
	}

	insertEntity := &model.UserRoleEntity{
		TenantID: req.TenantID,
		UserID:   req.UserID,
		RoleID:   req.RoleID,
		CreatedBy: gincontext.GetUserID(ctx),
	}

	if err := dao.NewUserRoleDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateUserRole] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserRoleCreateError)
	}
	return &dtopermission.UserRoleCreateResp{}, nil
}

func (svc *permissionSvc) DeleteUserRole(ctx *gin.Context, req *dtopermission.UserRoleDeleteReq) error {
	userRoleEntity, err := dao.NewUserRoleDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteUserRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleDeleteError)
	}
	if userRoleEntity == nil || userRoleEntity.ID == 0 {
		return code.GetError(code.UserRoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewUserRoleDao().Delete(ctx, req.UserID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteUserRole] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleDeleteError)
	}
	return nil
}

func (svc *permissionSvc) PageListUserRole(ctx *gin.Context, req *dtopermission.UserRolePageListReq) (*dtopermission.UserRolePageListResp, error) {
	cond := &dao.UserRoleCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		UserID:   req.UserID,
		RoleID:   req.RoleID,
	}
	userRoleEntityList, total, err := dao.NewUserRoleDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListUserRole] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserRoleGetPageListError)
	}

	list := make([]dtopermission.UserRolePageListItem, 0, len(userRoleEntityList))
	for _, v := range userRoleEntityList {
		list = append(list, dtopermission.UserRolePageListItem{
			UserID:   v.UserID,
			RoleID:   v.RoleID,
			TenantID: v.TenantID,
		})
	}
	return &dtopermission.UserRolePageListResp{
		List:  list,
		Total: total,
	}, nil
}
