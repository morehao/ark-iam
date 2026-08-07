package svcpermission

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type roleScopeDeleteRepository interface {
	GetListByCond(ctx context.Context, cond genericdao.Cond) (model.RoleScopeEntityList, error)
	Delete(ctx context.Context, id uint, userID uint) error
}

var newRoleScopeDeleteRepo = func() roleScopeDeleteRepository {
	return dao.NewRoleScopeDao()
}

type RoleScopeSvc interface {
	Create(ctx *gin.Context, req *dtopermission.RoleScopeCreateReq) (*dtopermission.RoleScopeCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.RoleScopeDeleteReq) error
	PageList(ctx *gin.Context, req *dtopermission.RoleScopePageListReq) (*dtopermission.RoleScopePageListResp, error)
}

type roleScopeSvc struct{}

var _ RoleScopeSvc = (*roleScopeSvc)(nil)

func NewRoleScopeSvc() RoleScopeSvc {
	return &roleScopeSvc{}
}

func (svc *roleScopeSvc) Create(ctx *gin.Context, req *dtopermission.RoleScopeCreateReq) (*dtopermission.RoleScopeCreateResp, error) {
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

func (svc *roleScopeSvc) Delete(ctx *gin.Context, req *dtopermission.RoleScopeDeleteReq) error {
	deleteRepo := newRoleScopeDeleteRepo()
	roleScopeEntityList, err := deleteRepo.GetListByCond(ctx, &dao.RoleScopeCond{
		TenantID: gincontext.GetTenantID(ctx),
		RoleID:   req.RoleID,
		ScopeID:  req.ScopeID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleScope] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleScopeDeleteError)
	}
	if len(roleScopeEntityList) == 0 || roleScopeEntityList[0].ID == 0 {
		return code.GetError(code.RoleScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := deleteRepo.Delete(ctx, roleScopeEntityList[0].ID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRoleScope] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleScopeDeleteError)
	}
	return nil
}

func (svc *roleScopeSvc) PageList(ctx *gin.Context, req *dtopermission.RoleScopePageListReq) (*dtopermission.RoleScopePageListResp, error) {
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
