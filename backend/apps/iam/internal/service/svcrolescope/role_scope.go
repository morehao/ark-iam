package svcrolescope

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtorole"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type RoleScopeSvc interface {
	Create(ctx *gin.Context, req *dtorole.RoleScopeCreateReq) (*dtorole.RoleScopeCreateResp, error)
	Delete(ctx *gin.Context, req *dtorole.RoleScopeDeleteReq) error
	PageList(ctx *gin.Context, req *dtorole.RoleScopePageListReq) (*dtorole.RoleScopePageListResp, error)
}

type roleScopeSvc struct {
}

var _ RoleScopeSvc = (*roleScopeSvc)(nil)

func NewRoleScopeSvc() RoleScopeSvc {
	return &roleScopeSvc{}
}

func (svc *roleScopeSvc) Create(ctx *gin.Context, req *dtorole.RoleScopeCreateReq) (*dtorole.RoleScopeCreateResp, error) {
	roleDao := dao.NewRoleDao()
	roleEntity, err := roleDao.GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcrolescope.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return nil, code.GetError(code.RoleNotExistError)
	}

	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcrolescope.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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
		glog.Errorf(ctx, "[svcrolescope.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleScopeCreateError)
	}
	return &dtorole.RoleScopeCreateResp{}, nil
}

func (svc *roleScopeSvc) Delete(ctx *gin.Context, req *dtorole.RoleScopeDeleteReq) error {
	roleScopeEntity, err := dao.NewRoleScopeDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcrolescope.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleScopeDeleteError)
	}
	if roleScopeEntity == nil || roleScopeEntity.ID == 0 {
		return code.GetError(code.RoleScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewRoleScopeDao().Delete(ctx, req.RoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcrolescope.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleScopeDeleteError)
	}
	return nil
}

func (svc *roleScopeSvc) PageList(ctx *gin.Context, req *dtorole.RoleScopePageListReq) (*dtorole.RoleScopePageListResp, error) {
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
		glog.Errorf(ctx, "[svcrolescope.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleScopeGetPageListError)
	}

	list := make([]dtorole.RoleScopePageListItem, 0, len(roleScopeEntityList))
	for _, v := range roleScopeEntityList {
		list = append(list, dtorole.RoleScopePageListItem{
			RoleID:   v.RoleID,
			ScopeID:  v.ScopeID,
			TenantID: v.TenantID,
		})
	}
	return &dtorole.RoleScopePageListResp{
		List:  list,
		Total: total,
	}, nil
}