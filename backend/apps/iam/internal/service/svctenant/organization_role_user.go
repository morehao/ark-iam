package svctenant

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type organizationRoleUserDeleteRepository interface {
	GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationRoleUserEntityList, error)
	Delete(ctx context.Context, id uint, userID uint) error
}

var newOrganizationRoleUserDeleteRepo = func() organizationRoleUserDeleteRepository {
	return dao.NewOrganizationRoleUserDao()
}

type OrganizationRoleUserSvc interface {
	Create(ctx *gin.Context, req *dtotenant.OrganizationRoleUserCreateReq) (*dtotenant.OrganizationRoleUserCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.OrganizationRoleUserDeleteReq) error
	PageList(ctx *gin.Context, req *dtotenant.OrganizationRoleUserPageListReq) (*dtotenant.OrganizationRoleUserPageListResp, error)
}

type organizationRoleUserSvc struct {
}

var _ OrganizationRoleUserSvc = (*organizationRoleUserSvc)(nil)

func NewOrganizationRoleUserSvc() OrganizationRoleUserSvc {
	return &organizationRoleUserSvc{}
}

func (svc *organizationRoleUserSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationRoleUserCreateReq) (*dtotenant.OrganizationRoleUserCreateResp, error) {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuser.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleGetDetailError)
	}
	if orgRoleEntity == nil || orgRoleEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationRoleNotExistError)
	}

	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuser.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	insertEntity := &model.OrganizationRoleUserEntity{
		TenantID:           gincontext.GetTenantID(ctx),
		OrganizationID:     req.OrganizationID,
		OrganizationRoleID: req.OrganizationRoleID,
		UserID:             req.UserID,
		CreatedBy:          gincontext.GetUserID(ctx),
	}

	if err := dao.NewOrganizationRoleUserDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuser.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleUserCreateError)
	}
	return &dtotenant.OrganizationRoleUserCreateResp{}, nil
}

func (svc *organizationRoleUserSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationRoleUserDeleteReq) error {
	deleteRepo := newOrganizationRoleUserDeleteRepo()
	orgRoleUserEntityList, err := deleteRepo.GetListByCond(ctx, &dao.OrganizationRoleUserCond{
		TenantID:           gincontext.GetTenantID(ctx),
		OrganizationRoleID: req.OrganizationRoleID,
		UserID:             req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuser.Delete] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUserDeleteError)
	}
	if len(orgRoleUserEntityList) == 0 || orgRoleUserEntityList[0].ID == 0 {
		return code.GetError(code.OrganizationRoleUserNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := deleteRepo.Delete(ctx, orgRoleUserEntityList[0].ID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuser.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUserDeleteError)
	}
	return nil
}

func (svc *organizationRoleUserSvc) PageList(ctx *gin.Context, req *dtotenant.OrganizationRoleUserPageListReq) (*dtotenant.OrganizationRoleUserPageListResp, error) {
	cond := &dao.OrganizationRoleUserCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:           req.TenantID,
		OrganizationID:     req.OrganizationID,
		OrganizationRoleID: req.OrganizationRoleID,
		UserID:             req.UserID,
	}
	orgRoleUserEntityList, total, err := dao.NewOrganizationRoleUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleUserGetPageListError)
	}

	list := make([]dtotenant.OrganizationRoleUserPageListItem, 0, len(orgRoleUserEntityList))
	for _, v := range orgRoleUserEntityList {
		list = append(list, dtotenant.OrganizationRoleUserPageListItem{
			OrganizationID:     v.OrganizationID,
			OrganizationRoleID: v.OrganizationRoleID,
			UserID:             v.UserID,
			TenantID:           v.TenantID,
		})
	}
	return &dtotenant.OrganizationRoleUserPageListResp{
		List:  list,
		Total: total,
	}, nil
}
