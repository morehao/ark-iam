package svcuser

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcperson"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
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

type delegatedPersonIdentitySvc interface {
	Create(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error)
	Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error
	Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error
	Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error)
	PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error)
	GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error)
}

type userIdentityUserResolver interface {
	GetByID(ctx context.Context, id uint) (*model.UserEntity, error)
}

type userIdentityRepository interface {
	Insert(ctx context.Context, entity *model.UserIdentityEntity) error
	GetByID(ctx context.Context, id uint) (*model.UserIdentityEntity, error)
	Delete(ctx context.Context, id uint, deletedBy uint) error
	UpdateMap(ctx context.Context, id uint, updates map[string]any) error
	GetPageListByCond(ctx context.Context, cond *dao.UserIdentityCond) (model.UserIdentityEntityList, int64, error)
}

var newUserIdentityRepo = func() userIdentityRepository {
	return &userIdentityRepoAdapter{dao: dao.NewUserIdentityDao()}
}

var newPersonIdentitySvc = func() delegatedPersonIdentitySvc {
	return svcperson.NewPersonSvc()
}

var newUserIdentityUserRepo = func() userIdentityUserResolver {
	return &userIdentityUserRepoAdapter{dao: dao.NewUserDao()}
}

type userIdentityRepoAdapter struct {
	dao *dao.UserIdentityDao
}

type userIdentityUserRepoAdapter struct {
	dao *dao.UserDao
}

func (r *userIdentityRepoAdapter) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	return r.dao.Insert(ctx, entity)
}

func (r *userIdentityRepoAdapter) GetByID(ctx context.Context, id uint) (*model.UserIdentityEntity, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *userIdentityRepoAdapter) Delete(ctx context.Context, id uint, deletedBy uint) error {
	return r.dao.Delete(ctx, id, deletedBy)
}

func (r *userIdentityRepoAdapter) UpdateMap(ctx context.Context, id uint, updates map[string]any) error {
	return r.dao.UpdateMap(ctx, id, updates)
}

func (r *userIdentityRepoAdapter) GetPageListByCond(ctx context.Context, cond *dao.UserIdentityCond) (model.UserIdentityEntityList, int64, error) {
	return r.dao.GetPageListByCond(ctx, cond)
}

func (r *userIdentityUserRepoAdapter) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	return r.dao.GetByID(ctx, id)
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
	return newPersonIdentitySvc().Create(ctx, mappedReq)
}

func (svc *userIdentitySvc) Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error {
	return newPersonIdentitySvc().Delete(ctx, req)
}

func (svc *userIdentitySvc) Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error {
	mappedReq, err := svc.mapUserIdentityUpdateReqToPerson(ctx, req)
	if err != nil {
		return err
	}
	return newPersonIdentitySvc().Update(ctx, mappedReq)
}

func (svc *userIdentitySvc) Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error) {
	if _, err := svc.resolveTenantUser(ctx, req.UserIdentityID); err != nil {
		return nil, err
	}
	return newPersonIdentitySvc().Detail(ctx, req)
}

func (svc *userIdentitySvc) PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error) {
	return newPersonIdentitySvc().PageList(ctx, req)
}

func (svc *userIdentitySvc) GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error) {
	userEntity, err := svc.resolveTenantUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	return newPersonIdentitySvc().GetByUser(ctx, &dtouser.UserIdentityByUserReq{UserID: userEntity.PersonID})
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
	userEntity, err := newUserIdentityUserRepo().GetByID(ctx, userID)
	if err != nil {
		return nil, code.GetError(code.UserNotExistError)
	}
	if userEntity == nil || userEntity.ID == 0 || userEntity.TenantID != gincontext.GetTenantID(ctx) || userEntity.PersonID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}
	return userEntity, nil
}
