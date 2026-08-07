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

type userRoleDeleteRepository interface {
	GetListByCond(ctx context.Context, cond gormdao.Cond) (model.UserRoleEntityList, error)
	Delete(ctx context.Context, id uint, userID uint) error
}

type userRoleCreateScopeRepository interface {
	GetRoleByID(ctx context.Context, id uint) (*model.RoleEntity, error)
	Insert(ctx context.Context, entity *model.UserRoleEntity) error
}

var newUserRoleDeleteRepo = func() userRoleDeleteRepository {
	return dao.NewUserRoleDao()
}

var newUserRoleCreateScopeRepo = func() userRoleCreateScopeRepository {
	return &userRoleCreateScopeDAO{}
}

type userRoleCreateScopeDAO struct{}

func (d *userRoleCreateScopeDAO) GetRoleByID(ctx context.Context, id uint) (*model.RoleEntity, error) {
	return dao.NewRoleDao().GetByID(ctx, id)
}

func (d *userRoleCreateScopeDAO) Insert(ctx context.Context, entity *model.UserRoleEntity) error {
	return dao.NewUserRoleDao().Insert(ctx, entity)
}

type UserRoleSvc interface {
	Create(ctx *gin.Context, req *dtopermission.UserRoleCreateReq) (*dtopermission.UserRoleCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.UserRoleDeleteReq) error
	PageList(ctx *gin.Context, req *dtopermission.UserRolePageListReq) (*dtopermission.UserRolePageListResp, error)
}

type userRoleSvc struct{}

var _ UserRoleSvc = (*userRoleSvc)(nil)

func NewUserRoleSvc() UserRoleSvc {
	return &userRoleSvc{}
}

func (svc *userRoleSvc) Create(ctx *gin.Context, req *dtopermission.UserRoleCreateReq) (*dtopermission.UserRoleCreateResp, error) {
	repo := newUserRoleCreateScopeRepo()
	roleEntity, err := repo.GetRoleByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateUserRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if !roleVisibleToTenant(roleEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.RoleNotExistError)
	}

	insertEntity := &model.UserRoleEntity{
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		RoleID:    req.RoleID,
		CreatedBy: gincontext.GetUserID(ctx),
	}

	if err := repo.Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateUserRole] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserRoleCreateError)
	}
	return &dtopermission.UserRoleCreateResp{}, nil
}

func (svc *userRoleSvc) Delete(ctx *gin.Context, req *dtopermission.UserRoleDeleteReq) error {
	deleteRepo := newUserRoleDeleteRepo()
	userRoleEntityList, err := deleteRepo.GetListByCond(ctx, &dao.UserRoleCond{
		TenantID: gincontext.GetTenantID(ctx),
		UserID:   req.UserID,
		RoleID:   req.RoleID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteUserRole] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleDeleteError)
	}
	if len(userRoleEntityList) == 0 || userRoleEntityList[0].ID == 0 {
		return code.GetError(code.UserRoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := deleteRepo.Delete(ctx, userRoleEntityList[0].ID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteUserRole] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleDeleteError)
	}
	return nil
}

func (svc *userRoleSvc) PageList(ctx *gin.Context, req *dtopermission.UserRolePageListReq) (*dtopermission.UserRolePageListResp, error) {
	cond := &dao.UserRoleCond{
		BaseCond: &gormdao.BaseCond{
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
