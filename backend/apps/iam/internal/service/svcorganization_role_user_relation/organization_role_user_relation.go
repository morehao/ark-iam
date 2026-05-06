package svcorganization_role_user_relation

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

type OrganizationRoleUserRelationSvc interface {
	Create(ctx *gin.Context, req *dtoorganization.OrganizationRoleUserRelationCreateReq) (*dtoorganization.OrganizationRoleUserRelationCreateResp, error)
	Delete(ctx *gin.Context, req *dtoorganization.OrganizationRoleUserRelationDeleteReq) error
	PageList(ctx *gin.Context, req *dtoorganization.OrganizationRoleUserRelationPageListReq) (*dtoorganization.OrganizationRoleUserRelationPageListResp, error)
}

type organizationRoleUserRelationSvc struct {
}

var _ OrganizationRoleUserRelationSvc = (*organizationRoleUserRelationSvc)(nil)

func NewOrganizationRoleUserRelationSvc() OrganizationRoleUserRelationSvc {
	return &organizationRoleUserRelationSvc{}
}

func (svc *organizationRoleUserRelationSvc) Create(ctx *gin.Context, req *dtoorganization.OrganizationRoleUserRelationCreateReq) (*dtoorganization.OrganizationRoleUserRelationCreateResp, error) {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization_role_user_relation.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleGetDetailError)
	}
	if orgRoleEntity == nil || orgRoleEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationRoleNotExistError)
	}

	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization_role_user_relation.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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
		glog.Errorf(ctx, "[svcorganization_role_user_relation.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleUserRelationCreateError)
	}
	return &dtoorganization.OrganizationRoleUserRelationCreateResp{}, nil
}

func (svc *organizationRoleUserRelationSvc) Delete(ctx *gin.Context, req *dtoorganization.OrganizationRoleUserRelationDeleteReq) error {
	orgRoleUserEntity, err := dao.NewOrganizationRoleUserRelationDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization_role_user_relation.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUserRelationDeleteError)
	}
	if orgRoleUserEntity == nil || orgRoleUserEntity.ID == 0 {
		return code.GetError(code.OrganizationRoleUserRelationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewOrganizationRoleUserRelationDao().Delete(ctx, req.OrganizationRoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganization_role_user_relation.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUserRelationDeleteError)
	}
	return nil
}

func (svc *organizationRoleUserRelationSvc) PageList(ctx *gin.Context, req *dtoorganization.OrganizationRoleUserRelationPageListReq) (*dtoorganization.OrganizationRoleUserRelationPageListResp, error) {
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
		glog.Errorf(ctx, "[svcorganization_role_user_relation.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleUserRelationGetPageListError)
	}

	list := make([]dtoorganization.OrganizationRoleUserRelationPageListItem, 0, len(orgRoleUserEntityList))
	for _, v := range orgRoleUserEntityList {
		list = append(list, dtoorganization.OrganizationRoleUserRelationPageListItem{
			OrganizationID:     v.OrganizationID,
			OrganizationRoleID: v.OrganizationRoleID,
			UserID:             v.UserID,
			TenantID:           v.TenantID,
		})
	}
	return &dtoorganization.OrganizationRoleUserRelationPageListResp{
		List:  list,
		Total: total,
	}, nil
}