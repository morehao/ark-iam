package svctenant

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type organizationRoleUserRelationDeleteRepository interface {
	GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationRoleUserRelationEntityList, error)
	Delete(ctx context.Context, id uint, userID uint) error
}

var newOrganizationRoleUserRelationDeleteRepo = func() organizationRoleUserRelationDeleteRepository {
	return dao.NewOrganizationRoleUserRelationDao()
}

type OrganizationRoleUserRelationSvc interface {
	Create(ctx *gin.Context, req *dtotenant.OrganizationRoleUserRelationCreateReq) (*dtotenant.OrganizationRoleUserRelationCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.OrganizationRoleUserRelationDeleteReq) error
	PageList(ctx *gin.Context, req *dtotenant.OrganizationRoleUserRelationPageListReq) (*dtotenant.OrganizationRoleUserRelationPageListResp, error)
}

type organizationRoleUserRelationSvc struct {
}

var _ OrganizationRoleUserRelationSvc = (*organizationRoleUserRelationSvc)(nil)

func NewOrganizationRoleUserRelationSvc() OrganizationRoleUserRelationSvc {
	return &organizationRoleUserRelationSvc{}
}

func (svc *organizationRoleUserRelationSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationRoleUserRelationCreateReq) (*dtotenant.OrganizationRoleUserRelationCreateResp, error) {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuserrelation.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleGetDetailError)
	}
	if orgRoleEntity == nil || orgRoleEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationRoleNotExistError)
	}

	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuserrelation.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	insertEntity := &model.OrganizationRoleUserRelationEntity{
		TenantID:           req.TenantID,
		OrganizationID:     req.OrganizationID,
		OrganizationRoleID: req.OrganizationRoleID,
		UserID:             req.UserID,
		CreatedBy:          gincontext.GetUserID(ctx),
	}

	if err := dao.NewOrganizationRoleUserRelationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuserrelation.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleUserRelationCreateError)
	}
	return &dtotenant.OrganizationRoleUserRelationCreateResp{}, nil
}

func (svc *organizationRoleUserRelationSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationRoleUserRelationDeleteReq) error {
	deleteRepo := newOrganizationRoleUserRelationDeleteRepo()
	orgRoleUserEntityList, err := deleteRepo.GetListByCond(ctx, &dao.OrganizationRoleUserRelationCond{
		TenantID:           gincontext.GetTenantID(ctx),
		OrganizationRoleID: req.OrganizationRoleID,
		UserID:             req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuserrelation.Delete] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUserRelationDeleteError)
	}
	if len(orgRoleUserEntityList) == 0 || orgRoleUserEntityList[0].ID == 0 {
		return code.GetError(code.OrganizationRoleUserRelationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := deleteRepo.Delete(ctx, orgRoleUserEntityList[0].ID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuserrelation.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUserRelationDeleteError)
	}
	return nil
}

func (svc *organizationRoleUserRelationSvc) PageList(ctx *gin.Context, req *dtotenant.OrganizationRoleUserRelationPageListReq) (*dtotenant.OrganizationRoleUserRelationPageListResp, error) {
	cond := &dao.OrganizationRoleUserRelationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:           req.TenantID,
		OrganizationID:     req.OrganizationID,
		OrganizationRoleID: req.OrganizationRoleID,
		UserID:             req.UserID,
	}
	orgRoleUserEntityList, total, err := dao.NewOrganizationRoleUserRelationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationroleuserrelation.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleUserRelationGetPageListError)
	}

	list := make([]dtotenant.OrganizationRoleUserRelationPageListItem, 0, len(orgRoleUserEntityList))
	for _, v := range orgRoleUserEntityList {
		list = append(list, dtotenant.OrganizationRoleUserRelationPageListItem{
			OrganizationID:     v.OrganizationID,
			OrganizationRoleID: v.OrganizationRoleID,
			UserID:             v.UserID,
			TenantID:           v.TenantID,
		})
	}
	return &dtotenant.OrganizationRoleUserRelationPageListResp{
		List:  list,
		Total: total,
	}, nil
}
