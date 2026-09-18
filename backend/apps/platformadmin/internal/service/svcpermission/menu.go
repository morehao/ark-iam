package svcpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/menu"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

func menuVisible(entity *model.MenuEntity) bool {
	return entity != nil && entity.ID != ""
}

// validateMenuEnums 校验菜单字典枚举合法值：Type/Status/Visibility 必须命中白名单常量；
// Hidden/ExternalLink/KeepAlive 三个开关允许空串（＝调用方未提供，按 disable 归一）。
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

// normalizeMenuSwitches 把未提供的开关（空串）归一为 disable：列默认值即 disable，
// 而更新走 UpdateMap（map 不经过 GORM 的默认值省略），空串直写会落成脏枚举值；
// 归一后语义与原 bool 字段的 false（未勾选）完全一致。
func normalizeMenuSwitches(req *objpermission.MenuBaseInfo) {
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
	normalizeMenuSwitches(&req.MenuBaseInfo)
	if req.Code == "" {
		glog.Errorf(ctx, "[svcpermission.CreateMenu] 菜单编码不得为空, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	// 菜单可挂到任意应用（含内置应用）：控制台自建行 seed_key 恒为空，种子不会认领它；
	// 内置菜单的删除则靠软删"墓碑"保证持久生效（见 pkg/seed.menuSeedKeyRemoved）。
	// 因此"按需扩展/调整菜单"不再需要改代码发版。
	ok, err := menuParentUsable(ctx, req.AppID, req.ParentID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateMenu] 校验上级菜单失败, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	if !ok {
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
	// 级联删除整棵子树：子菜单一旦失去父级就再也进不了应用菜单树（树从 parent_id="" 构建），
	// 只会变成看不见又删不掉的孤儿行，因此删除父级必须连同子孙一起下线。
	subtreeIDs, err := collectMenuSubtreeIDs(ctx, menuEntity.AppID, menuEntity.ID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteMenu] 收集子树失败, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, menuID := range subtreeIDs {
			// 先解绑角色授权，否则 role_menu 会留下指向已删除菜单的悬空绑定
			bindings, err := dao.NewRoleMenuDao().WithTx(tx).GetListByCond(ctx, &dao.RoleMenuCond{MenuID: menuID})
			if err != nil {
				return err
			}
			for _, binding := range bindings {
				if err := dao.NewRoleMenuDao().WithTx(tx).Delete(ctx, binding.ID, userID); err != nil {
					return err
				}
			}
			if err := dao.NewMenuDao().WithTx(tx).Delete(ctx, menuID, userID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteMenu] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	return nil
}

// menuParentUsable 校验上级菜单：空串表示根菜单（合法）；否则父级必须存在且与本菜单同属一个应用。
// 不校验会让子菜单挂到不存在或别的应用的父级下——它不会出现在任何应用菜单树里，成为看不见的孤儿行。
func menuParentUsable(ctx *gin.Context, appID, parentID string) (bool, error) {
	if parentID == "" {
		return true, nil
	}
	parent, err := dao.NewMenuDao().GetByID(ctx, parentID)
	if err != nil {
		return false, err
	}
	if parent == nil || parent.ID == "" || parent.AppID != appID {
		return false, nil
	}
	return true, nil
}

// collectMenuSubtreeIDs 返回 rootID 及其全部子孙菜单 ID（限定同一应用）。
// 菜单树规模有限（单应用几十行），一次取回后内存遍历，避免逐层查询。
func collectMenuSubtreeIDs(ctx *gin.Context, appID, rootID string) ([]string, error) {
	list, err := dao.NewMenuDao().GetListByCond(ctx, &dao.MenuCond{AppID: appID})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.collectMenuSubtreeIDs] dao GetListByCond fail, err:%v, appID:%s", err, appID)
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
	normalizeMenuSwitches(&req.MenuBaseInfo)
	// 菜单编码可改（含内置菜单）：种子按 seed_key 认行，改 code/app_id 都不会导致重建行。
	// 仅要求非空——应用内唯一由唯一索引 uk_menu_app_code_active 兜底（撞重返回本领域更新错误码）。
	if req.Code == "" {
		glog.Errorf(ctx, "[svcpermission.UpdateMenu] 菜单编码不得为空, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	// 上级菜单校验：父级必须存在且与目标应用一致（否则子菜单进不了任何应用菜单树），
	// 且不得是自身或其子孙——成环后整棵子树都从菜单树里消失，等于自删。
	ok, err := menuParentUsable(ctx, req.AppID, req.ParentID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateMenu] 校验上级菜单失败, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	if !ok || req.ParentID == req.MenuID {
		return code.GetError(code.MenuUpdateError)
	}
	if req.ParentID != "" {
		subtreeIDs, err := collectMenuSubtreeIDs(ctx, menuEntity.AppID, req.MenuID)
		if err != nil {
			glog.Errorf(ctx, "[svcpermission.UpdateMenu] 校验菜单环失败, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.MenuUpdateError)
		}
		for _, id := range subtreeIDs {
			if id == req.ParentID {
				return code.GetError(code.MenuUpdateError)
			}
		}
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

func (svc *menuSvc) Tree(ctx *gin.Context, req *dtopermission.MenuTreeReq) (*dtopermission.MenuTreeResp, error) {
	menuRepo := dao.NewMenuDao()
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

	nodes, err := menu.BuildMyMenuTree(ctx, tenantID, userID, []string{appList[0].ID})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.MyTree] build my menu tree fail, err:%v", err)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	return &dtopermission.MenuMyTreeResp{List: nodes}, nil
}
