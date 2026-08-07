package svcpermission

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type roleMenuDeleteRepository interface {
	GetListByCond(ctx context.Context, cond gormdao.Cond) (model.RoleMenuEntityList, error)
	Delete(ctx context.Context, id uint, userID uint) error
}

var newRoleMenuDeleteRepo = func() roleMenuDeleteRepository {
	return dao.NewRoleMenuDao()
}

type RoleMenuSvc interface {
	Create(ctx *gin.Context, req *dtopermission.RoleMenuCreateReq) (*dtopermission.RoleMenuCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.RoleMenuDeleteReq) error
	PageList(ctx *gin.Context, req *dtopermission.RoleMenuPageListReq) (*dtopermission.RoleMenuPageListResp, error)
}

type roleMenuSvc struct{}

var _ RoleMenuSvc = (*roleMenuSvc)(nil)

func NewRoleMenuSvc() RoleMenuSvc {
	return &roleMenuSvc{}
}

func (svc *roleMenuSvc) Create(ctx *gin.Context, req *dtopermission.RoleMenuCreateReq) (*dtopermission.RoleMenuCreateResp, error) {
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
		TenantID:  req.TenantID,
		RoleID:    req.RoleID,
		MenuID:    req.MenuID,
		CreatedBy: gincontext.GetUserID(ctx),
	}

	if err := dao.NewRoleMenuDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRoleMenu] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleMenuCreateError)
	}
	return &dtopermission.RoleMenuCreateResp{}, nil
}

func (svc *roleMenuSvc) Delete(ctx *gin.Context, req *dtopermission.RoleMenuDeleteReq) error {
	deleteRepo := newRoleMenuDeleteRepo()
	roleMenuEntityList, err := deleteRepo.GetListByCond(ctx, &dao.RoleMenuCond{
		TenantID: gincontext.GetTenantID(ctx),
		RoleID:   req.RoleID,
		MenuID:   req.MenuID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleMenu] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleMenuDeleteError)
	}
	if len(roleMenuEntityList) == 0 || roleMenuEntityList[0].ID == 0 {
		return code.GetError(code.RoleMenuNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := deleteRepo.Delete(ctx, roleMenuEntityList[0].ID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleMenu] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleMenuDeleteError)
	}
	return nil
}

func (svc *roleMenuSvc) PageList(ctx *gin.Context, req *dtopermission.RoleMenuPageListReq) (*dtopermission.RoleMenuPageListResp, error) {
	cond := &dao.RoleMenuCond{
		BaseCond: &gormdao.BaseCond{
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
