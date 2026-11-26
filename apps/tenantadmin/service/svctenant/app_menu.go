package svctenant

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/application"
	"github.com/morehao/ark-iam/pkg/core/menu"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

// AppMenuSvc 租户侧「应用菜单」维护服务：租户只能维护自己归属（owner_type=tenant 且
// owner_tenant_id==当前租户）的应用菜单。与 TenantMenuSvc（只读侧栏树）分离。
//
// 安全要点（iam_menu / iam_application 均为全局表，租户插件不注入 tenant_id）：
//  1. 每个写/管理读方法入口先 requireSystemAdmin（admin_type=admin）；
//  2. AssertCurrentTenantScope 证明当前是"具体当前租户"作用域（fail-closed，防 AllScope 误判）；
//  3. LoadOwnedApp 证明目标 app 归属当前租户（非 owner/不存在 → MenuAppNotOwnedError）；
//  4. 写落到 DAO 的 app_id 约束方法（UpdateMapScoped/DeleteScoped），authorizedAppID 来自第 3 步
//     的 app.ID，绝不取请求体 appID —— 即便 service 判错也写不进别人应用的菜单。
type AppMenuSvc interface {
	Create(ctx context.Context, req *dtotenant.AppMenuCreateReq) (*dtotenant.AppMenuCreateResp, error)
	Update(ctx context.Context, req *dtotenant.AppMenuUpdateReq) error
	Delete(ctx context.Context, req *dtotenant.AppMenuDeleteReq) error
	Detail(ctx context.Context, req *dtotenant.AppMenuDetailReq) (*dtotenant.MenuTreeItem, error)
	Tree(ctx context.Context, req *dtotenant.AppMenuTreeReq) (*dtotenant.MenuTreeResp, error)
}

type appMenuSvc struct {
	db gormdao.DBGetter
}

var _ AppMenuSvc = (*appMenuSvc)(nil)

func NewAppMenuSvc(db gormdao.DBGetter) AppMenuSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &appMenuSvc{db: db}
}

// authorizeApp 校验当前操作人可维护 appID 的菜单，返回权威 appID（来自 LoadOwnedApp）。
func (svc *appMenuSvc) authorizeApp(ctx context.Context, operator objidentity.Operator, appID string, opErr int) (string, error) {
	// 1) 系统管理员硬门槛（与部门/角色/密钥等管理写一致）
	if err := requireSystemAdmin(ctx, operator, opErr); err != nil {
		return "", err
	}
	// 2) 具体当前租户作用域证明（fail-closed）
	tenantID, err := application.AssertCurrentTenantScope(ctx)
	if err != nil {
		glog.Errorf(ctx, "[appMenuSvc.authorizeApp] 无具体租户作用域, appID:%s, err:%v", appID, err)
		return "", code.GetError(code.MenuAppNotOwnedError)
	}
	// 3) 应用归属校验（非 owner/不存在 → 统一 MenuAppNotOwnedError，防枚举）
	app, err := application.LoadOwnedApp(ctx, appID, tenantID)
	if err != nil {
		if err == application.ErrAppNotOwned {
			return "", code.GetError(code.MenuAppNotOwnedError)
		}
		glog.Errorf(ctx, "[appMenuSvc.authorizeApp] LoadOwnedApp fail, appID:%s, err:%v", appID, err)
		return "", code.GetError(opErr)
	}
	return app.ID, nil
}


// fieldsToBaseInfo 把租户侧可写字段转为公共 MenuBaseInfo 以复用枚举校验/归一化工具（AppID 不参与校验）。
func fieldsToBaseInfo(f *dtotenant.AppMenuFields) objpermission.MenuBaseInfo {
	return objpermission.MenuBaseInfo{
		ParentID:     f.ParentID,
		Name:         f.Name,
		Code:         f.Code,
		Path:         f.Path,
		Icon:         f.Icon,
		Sort:         f.Sort,
		Type:         f.Type,
		Visibility:   f.Visibility,
		Component:    f.Component,
		Redirect:     f.Redirect,
		Hidden:       f.Hidden,
		ExternalLink: f.ExternalLink,
		KeepAlive:    f.KeepAlive,
		Status:       f.Status,
	}
}

// baseInfoToFields 把校验/归一化后的公共字段回写到租户侧可写字段（switch 归一值需要回写）。
func baseInfoToFields(b *objpermission.MenuBaseInfo, f *dtotenant.AppMenuFields) {
	f.Hidden = b.Hidden
	f.ExternalLink = b.ExternalLink
	f.KeepAlive = b.KeepAlive
	f.Visibility = b.Visibility
	f.Status = b.Status
	f.Type = b.Type
}

func (svc *appMenuSvc) Create(ctx context.Context, req *dtotenant.AppMenuCreateReq) (*dtotenant.AppMenuCreateResp, error) {
	// 请求体里的 MenuBaseInfo.AppID 不可信，用 path 的 req.AppID 校验归属
	authorizedAppID, err := svc.authorizeApp(ctx, req.Operator, req.AppID, code.MenuCreateError)
	if err != nil {
		return nil, err
	}
	baseInfo := fieldsToBaseInfo(&req.AppMenuFields)
	if !menu.ValidateMenuEnums(&baseInfo) {
		return nil, code.GetError(code.MenuCreateError)
	}
	menu.NormalizeMenuSwitches(&baseInfo)
	baseInfoToFields(&baseInfo, &req.AppMenuFields)
	if req.Code == "" {
		glog.Errorf(ctx, "[appMenuSvc.Create] 菜单编码不得为空, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	// 父菜单必须存在且同属该应用
	if _, err := menu.MenuParentUsable(svc.db, ctx, authorizedAppID, req.ParentID); err != nil {
		if err == menu.ErrParentUnusable {
			return nil, code.GetError(code.MenuCreateError)
		}
		glog.Errorf(ctx, "[appMenuSvc.Create] 校验上级菜单失败, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	insertEntity := &model.MenuEntity{
		AppID:        authorizedAppID, // 强制用权威 appID，忽略 body 的 AppID
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
		CreatedBy:    req.Operator.UserID,
	}
	if err := dao.NewMenuDao(dao.WithDBGetter(svc.db)).Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[appMenuSvc.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuCreateError)
	}
	return &dtotenant.AppMenuCreateResp{MenuID: insertEntity.ID}, nil
}

func (svc *appMenuSvc) Update(ctx context.Context, req *dtotenant.AppMenuUpdateReq) error {
	authorizedAppID, err := svc.authorizeApp(ctx, req.Operator, req.AppID, code.MenuUpdateError)
	if err != nil {
		return err
	}
	// 目标菜单必须存在于该应用下（防用 A 的 appID 传 B 的 menuID）
	menuEntity, err := dao.NewMenuDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[appMenuSvc.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	if menuEntity == nil || menuEntity.ID == "" || menuEntity.AppID != authorizedAppID {
		return code.GetError(code.MenuNotExistError)
	}
	baseInfo := fieldsToBaseInfo(&req.AppMenuFields)
	if !menu.ValidateMenuEnums(&baseInfo) {
		return code.GetError(code.MenuUpdateError)
	}
	menu.NormalizeMenuSwitches(&baseInfo)
	baseInfoToFields(&baseInfo, &req.AppMenuFields)
	if req.Code == "" {
		glog.Errorf(ctx, "[appMenuSvc.Update] 菜单编码不得为空, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	// 父菜单校验：存在且同应用；不得是自身或子孙（成环）
	if _, err := menu.MenuParentUsable(svc.db, ctx, authorizedAppID, req.ParentID); err != nil {
		if err == menu.ErrParentUnusable {
			return code.GetError(code.MenuUpdateError)
		}
		glog.Errorf(ctx, "[appMenuSvc.Update] 校验上级菜单失败, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	if req.ParentID == req.MenuID {
		return code.GetError(code.MenuUpdateError)
	}
	if req.ParentID != "" {
		subtreeIDs, err := menu.CollectSubtreeIDs(svc.db, ctx, authorizedAppID, req.MenuID)
		if err != nil {
			glog.Errorf(ctx, "[appMenuSvc.Update] 校验菜单环失败, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.MenuUpdateError)
		}
		for _, id := range subtreeIDs {
			if id == req.ParentID {
				return code.GetError(code.MenuUpdateError)
			}
		}
	}
	// 不允许改 app_id：update map 不含 app_id
	updateMap := map[string]any{
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
		"updated_by":    req.Operator.UserID,
	}
	if err := dao.NewMenuDao(dao.WithDBGetter(svc.db)).UpdateMapScoped(ctx, req.MenuID, authorizedAppID, updateMap); err != nil {
		glog.Errorf(ctx, "[appMenuSvc.Update] dao UpdateMapScoped fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuUpdateError)
	}
	return nil
}

func (svc *appMenuSvc) Delete(ctx context.Context, req *dtotenant.AppMenuDeleteReq) error {
	authorizedAppID, err := svc.authorizeApp(ctx, req.Operator, req.AppID, code.MenuDeleteError)
	if err != nil {
		return err
	}
	menuEntity, err := dao.NewMenuDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[appMenuSvc.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	if menuEntity == nil || menuEntity.ID == "" || menuEntity.AppID != authorizedAppID {
		return code.GetError(code.MenuNotExistError)
	}
	// 级联删除整棵子树
	subtreeIDs, err := menu.CollectSubtreeIDs(svc.db, ctx, authorizedAppID, menuEntity.ID)
	if err != nil {
		glog.Errorf(ctx, "[appMenuSvc.Delete] 收集子树失败, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	userID := req.Operator.UserID
	txErr := svc.db(ctx).Transaction(func(tx *gorm.DB) error {
		// menu 是全局表、role_menu 是租户表：删菜单必须跨全部租户清理授权绑定（受控跨租户维护写，
		// 非业务读），否则会留下指向已删菜单的悬空授权。
		crossCtx := dbclient.CrossTenantContext(ctx)
		for _, menuID := range subtreeIDs {
			bindings, err := dao.NewRoleMenuDao(dao.WithDBGetter(svc.db)).WithTx(tx).GetListByCond(crossCtx, &dao.RoleMenuCond{MenuID: menuID})
			if err != nil {
				return err
			}
			for _, binding := range bindings {
				if err := dao.NewRoleMenuDao(dao.WithDBGetter(svc.db)).WithTx(tx).Delete(crossCtx, binding.ID, userID); err != nil {
					return err
				}
			}
			// app_id 约束的软删除（WHERE id=? AND app_id=?），即便前面判错也删不到别人应用的菜单。
			if err := tx.Model(&model.MenuEntity{}).Table(model.TableNameMenu).
				Where("id = ? AND app_id = ?", menuID, authorizedAppID).
				Updates(map[string]any{"deleted_at": time.Now(), "deleted_by": userID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[appMenuSvc.Delete] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.MenuDeleteError)
	}
	return nil
}

func (svc *appMenuSvc) Detail(ctx context.Context, req *dtotenant.AppMenuDetailReq) (*dtotenant.MenuTreeItem, error) {
	authorizedAppID, err := svc.authorizeApp(ctx, req.Operator, req.AppID, code.MenuGetDetailError)
	if err != nil {
		return nil, err
	}
	menuEntity, err := dao.NewMenuDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.MenuID)
	if err != nil {
		glog.Errorf(ctx, "[appMenuSvc.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MenuGetDetailError)
	}
	if menuEntity == nil || menuEntity.ID == "" || menuEntity.AppID != authorizedAppID {
		return nil, code.GetError(code.MenuNotExistError)
	}
	return &dtotenant.MenuTreeItem{
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
	}, nil
}

func (svc *appMenuSvc) Tree(ctx context.Context, req *dtotenant.AppMenuTreeReq) (*dtotenant.MenuTreeResp, error) {
	authorizedAppID, err := svc.authorizeApp(ctx, req.Operator, req.AppID, code.MenuGetPageListError)
	if err != nil {
		return nil, err
	}
	nodes, err := menu.BuildAppMenuTree(ctx, authorizedAppID)
	if err != nil {
		glog.Errorf(ctx, "[appMenuSvc.Tree] BuildAppMenuTree fail, err:%v, appID:%s", err, authorizedAppID)
		return nil, code.GetError(code.MenuGetPageListError)
	}
	return &dtotenant.MenuTreeResp{List: convertMenuNodes(nodes)}, nil
}
