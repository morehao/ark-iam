package svctenant

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type organizationUserDeleteRepository interface {
	GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationUserEntityList, error)
	Delete(ctx context.Context, id uint, userID uint) error
}

var newOrganizationUserDeleteRepo = func() organizationUserDeleteRepository {
	return dao.NewOrganizationUserDao()
}

type OrganizationUserSvc interface {
	Create(ctx *gin.Context, req *dtotenant.OrganizationUserCreateReq) (*dtotenant.OrganizationUserCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.OrganizationUserDeleteReq) error
	PageList(ctx *gin.Context, req *dtotenant.OrganizationUserPageListReq) (*dtotenant.OrganizationUserPageListResp, error)
}

type organizationUserSvc struct {
}

var _ OrganizationUserSvc = (*organizationUserSvc)(nil)

func NewOrganizationUserSvc() OrganizationUserSvc {
	return &organizationUserSvc{}
}

func (svc *organizationUserSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationUserCreateReq) (*dtotenant.OrganizationUserCreateResp, error) {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetDetailError)
	}
	if orgEntity == nil || orgEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	insertEntity := &model.OrganizationUserEntity{
		TenantID:       gincontext.GetTenantID(ctx),
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		CreatedBy:      gincontext.GetUserID(ctx),
	}

	if err := dao.NewOrganizationUserDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserCreateError)
	}
	return &dtotenant.OrganizationUserCreateResp{}, nil
}

func (svc *organizationUserSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationUserDeleteReq) error {
	deleteRepo := newOrganizationUserDeleteRepo()
	orgUserEntityList, err := deleteRepo.GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:       gincontext.GetTenantID(ctx),
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Delete] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserDeleteError)
	}
	if len(orgUserEntityList) == 0 || orgUserEntityList[0].ID == 0 {
		return code.GetError(code.OrganizationUserNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := deleteRepo.Delete(ctx, orgUserEntityList[0].ID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserDeleteError)
	}
	return nil
}

func (svc *organizationUserSvc) PageList(ctx *gin.Context, req *dtotenant.OrganizationUserPageListReq) (*dtotenant.OrganizationUserPageListResp, error) {
	cond := &dao.OrganizationUserCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
	}
	orgUserEntityList, total, err := dao.NewOrganizationUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserGetPageListError)
	}

	list := make([]dtotenant.OrganizationUserPageListItem, 0, len(orgUserEntityList))
	for _, v := range orgUserEntityList {
		list = append(list, dtotenant.OrganizationUserPageListItem{
			OrganizationID: v.OrganizationID,
			UserID:        v.UserID,
			TenantID:      v.TenantID,
		})
	}
	return &dtotenant.OrganizationUserPageListResp{
		List:  list,
		Total: total,
	}, nil
}
