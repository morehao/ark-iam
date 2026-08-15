package svcuser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcperson"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type UserIdentitySvc interface {
	Create(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error)
	Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error
	Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error
	Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error)
	PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error)
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

func (svc *userIdentitySvc) Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error {
	mappedReq, err := svc.mapUserIdentityUpdateReqToPerson(ctx, req)
	if err != nil {
		return err
	}
	return svcperson.NewPersonSvc().Update(ctx, mappedReq)
}

func (svc *userIdentitySvc) Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error) {
	// 直接委托 person 身份服务：其 Detail 已按 identity.PersonID 做租户可见性校验，
	// 此处不再用 resolveTenantUser（它按 userID 解析，把 identityID 当 userID 查是错位逻辑）。
	return svcperson.NewPersonSvc().Detail(ctx, req)
}

func (svc *userIdentitySvc) PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error) {
	return svcperson.NewPersonSvc().PageList(ctx, req)
}

func (svc *userIdentitySvc) GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error) {
	userEntity, err := svc.resolveTenantUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	return svcperson.NewPersonSvc().GetByUser(ctx, &dtouser.UserIdentityByUserReq{UserID: userEntity.PersonID})
}

func (svc *userIdentitySvc) mapUserIdentityReqToPerson(ctx *gin.Context, userID uint, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateReq, error) {
	userEntity, err := svc.resolveTenantUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	clone := *req
	clone.UserID = userEntity.PersonID
	clone.TenantID = gincontext.GetTenantID(ctx)
	return &clone, nil
}

func (svc *userIdentitySvc) mapUserIdentityUpdateReqToPerson(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) (*dtouser.UserIdentityUpdateReq, error) {
	clone := *req
	clone.TenantID = gincontext.GetTenantID(ctx)
	if req.UserID != 0 {
		userEntity, err := svc.resolveTenantUser(ctx, req.UserID)
		if err != nil {
			return nil, err
		}
		clone.UserID = userEntity.PersonID
	}
	return &clone, nil
}

func (svc *userIdentitySvc) resolveTenantUser(ctx *gin.Context, userID uint) (*model.UserEntity, error) {
	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil {
		return nil, code.GetError(code.UserNotExistError)
	}
	if userEntity == nil || userEntity.ID == 0 || userEntity.TenantID != gincontext.GetTenantID(ctx) || userEntity.PersonID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}
	return userEntity, nil
}
