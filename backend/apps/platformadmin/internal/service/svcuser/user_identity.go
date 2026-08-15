package svcuser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcperson"
)

type UserIdentitySvc interface {
	Create(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error)
	Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error
	GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error)
}

type userIdentitySvc struct {
}

var _ UserIdentitySvc = (*userIdentitySvc)(nil)

func NewUserIdentitySvc() UserIdentitySvc {
	return &userIdentitySvc{}
}

func (svc *userIdentitySvc) Create(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error) {
	mappedReq, err := svc.mapUserIdentityReqToPerson(ctx, req.UserID, req)
	if err != nil {
		return nil, err
	}
	return svcperson.NewPersonSvc().Create(ctx, mappedReq)
}

func (svc *userIdentitySvc) Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error {
	return svcperson.NewPersonSvc().Delete(ctx, req)
}

func (svc *userIdentitySvc) GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error) {
	userEntity, err := svc.resolveTenantUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	return svcperson.NewPersonSvc().GetByUser(ctx, &dtouser.UserIdentityByUserReq{UserID: userEntity.PersonID})
}

func (svc *userIdentitySvc) mapUserIdentityReqToPerson(ctx *gin.Context, userID string, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateReq, error) {
	userEntity, err := svc.resolveTenantUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	clone := *req
	clone.UserID = userEntity.PersonID
	clone.TenantID = gctx.GetTenantID(ctx)
	return &clone, nil
}

func (svc *userIdentitySvc) resolveTenantUser(ctx *gin.Context, userID string) (*model.UserEntity, error) {
	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil {
		return nil, code.GetError(code.UserNotExistError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gctx.GetTenantID(ctx) || userEntity.PersonID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}
	return userEntity, nil
}
