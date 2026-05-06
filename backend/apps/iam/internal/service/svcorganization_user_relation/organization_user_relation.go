package svcorganization_user_relation

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoorganization"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type OrganizationUserRelationSvc interface {
	Create(ctx *gin.Context, req *dtoorganization.OrganizationUserRelationCreateReq) (*dtoorganization.OrganizationUserRelationCreateResp, error)
	Delete(ctx *gin.Context, req *dtoorganization.OrganizationUserRelationDeleteReq) error
	PageList(ctx *gin.Context, req *dtoorganization.OrganizationUserRelationPageListReq) (*dtoorganization.OrganizationUserRelationPageListResp, error)
}

type organizationUserRelationSvc struct {
}

var _ OrganizationUserRelationSvc = (*organizationUserRelationSvc)(nil)

func NewOrganizationUserRelationSvc() OrganizationUserRelationSvc {
	return &organizationUserRelationSvc{}
}

func (svc *organizationUserRelationSvc) Create(ctx *gin.Context, req *dtoorganization.OrganizationUserRelationCreateReq) (*dtoorganization.OrganizationUserRelationCreateResp, error) {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization_user_relation.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetDetailError)
	}
	if orgEntity == nil || orgEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization_user_relation.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	insertEntity := &model.OrganizationUserRelationEntity{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		CreatedBy:      gincontext.GetUserID(ctx),
	}

	if err := dao.NewOrganizationUserRelationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganization_user_relation.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserRelationCreateError)
	}
	return &dtoorganization.OrganizationUserRelationCreateResp{}, nil
}

func (svc *organizationUserRelationSvc) Delete(ctx *gin.Context, req *dtoorganization.OrganizationUserRelationDeleteReq) error {
	orgUserEntity, err := dao.NewOrganizationUserRelationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization_user_relation.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserRelationDeleteError)
	}
	if orgUserEntity == nil || orgUserEntity.ID == 0 {
		return code.GetError(code.OrganizationUserRelationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewOrganizationUserRelationDao().Delete(ctx, req.OrganizationID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganization_user_relation.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserRelationDeleteError)
	}
	return nil
}

func (svc *organizationUserRelationSvc) PageList(ctx *gin.Context, req *dtoorganization.OrganizationUserRelationPageListReq) (*dtoorganization.OrganizationUserRelationPageListResp, error) {
	cond := &dao.OrganizationUserRelationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
	}
	orgUserEntityList, total, err := dao.NewOrganizationUserRelationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization_user_relation.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserRelationGetPageListError)
	}

	list := make([]dtoorganization.OrganizationUserRelationPageListItem, 0, len(orgUserEntityList))
	for _, v := range orgUserEntityList {
		list = append(list, dtoorganization.OrganizationUserRelationPageListItem{
			OrganizationID: v.OrganizationID,
			UserID:        v.UserID,
			TenantID:      v.TenantID,
		})
	}
	return &dtoorganization.OrganizationUserRelationPageListResp{
		List:  list,
		Total: total,
	}, nil
}