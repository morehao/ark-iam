package svcuser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type UserRoleSvc interface {
	Create(ctx *gin.Context, req *dtopermission.UserRoleCreateReq) (*dtopermission.UserRoleCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.UserRoleDeleteReq) error
	PageList(ctx *gin.Context, req *dtopermission.UserRolePageListReq) (*dtopermission.UserRolePageListResp, error)
}

type userRoleSvc struct {
}

var _ UserRoleSvc = (*userRoleSvc)(nil)

func NewUserRoleSvc() UserRoleSvc {
	return &userRoleSvc{}
}

func (svc *userRoleSvc) Create(ctx *gin.Context, req *dtopermission.UserRoleCreateReq) (*dtopermission.UserRoleCreateResp, error) {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcuserrole.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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
		glog.Errorf(ctx, "[svcuserrole.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserRoleCreateError)
	}
	return &dtopermission.UserRoleCreateResp{}, nil
}

func (svc *userRoleSvc) Delete(ctx *gin.Context, req *dtopermission.UserRoleDeleteReq) error {
	userRoleEntity, err := dao.NewUserRoleDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuserrole.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleDeleteError)
	}
	if userRoleEntity == nil || userRoleEntity.ID == 0 {
		return code.GetError(code.UserRoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewUserRoleDao().Delete(ctx, req.UserID, userID); err != nil {
		glog.Errorf(ctx, "[svcuserrole.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleDeleteError)
	}
	return nil
}

func (svc *userRoleSvc) PageList(ctx *gin.Context, req *dtopermission.UserRolePageListReq) (*dtopermission.UserRolePageListResp, error) {
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
		glog.Errorf(ctx, "[svcuserrole.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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