package svcuser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func (svc *userSvc) CreateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationCreateReq) (*dtouser.UserDepartmentRelationCreateResp, error) {
	insertEntity := &model.UserDepartmentRelationEntity{
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		IsPrimary:    req.IsPrimary,
		CreatedBy:    gincontext.GetUserID(ctx),
	}

	if err := dao.NewUserDepartmentRelationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcuser.CreateUserDepartmentRelation] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationCreateError)
	}
	return &dtouser.UserDepartmentRelationCreateResp{
		UserDepartmentRelationID: insertEntity.ID,
	}, nil
}

func (svc *userSvc) DeleteUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDeleteReq) error {
	userDepartmentRelationEntity, err := dao.NewUserDepartmentRelationDao().GetByID(ctx, req.UserDepartmentRelationID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.DeleteUserDepartmentRelation] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationDeleteError)
	}
	if userDepartmentRelationEntity == nil || userDepartmentRelationEntity.ID == 0 {
		return code.GetError(code.UserDepartmentRelationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewUserDepartmentRelationDao().Delete(ctx, req.UserDepartmentRelationID, userID); err != nil {
		glog.Errorf(ctx, "[svcuser.DeleteUserDepartmentRelation] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationDeleteError)
	}
	return nil
}

func (svc *userSvc) UpdateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationUpdateReq) error {
	userDepartmentRelationEntity, err := dao.NewUserDepartmentRelationDao().GetByID(ctx, req.UserDepartmentRelationID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateUserDepartmentRelation] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationUpdateError)
	}
	if userDepartmentRelationEntity == nil || userDepartmentRelationEntity.ID == 0 {
		return code.GetError(code.UserDepartmentRelationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":     req.TenantID,
		"user_id":       req.UserID,
		"department_id": req.DepartmentID,
		"is_primary":    req.IsPrimary,
		"updated_by":    userID,
	}
	if err := dao.NewUserDepartmentRelationDao().UpdateMap(ctx, req.UserDepartmentRelationID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateUserDepartmentRelation] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationUpdateError)
	}
	return nil
}

func (svc *userSvc) DetailUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDetailReq) (*dtouser.UserDepartmentRelationDetailResp, error) {
	userDepartmentRelationEntity, err := dao.NewUserDepartmentRelationDao().GetByID(ctx, req.UserDepartmentRelationID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.DetailUserDepartmentRelation] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationGetDetailError)
	}
	if userDepartmentRelationEntity == nil || userDepartmentRelationEntity.ID == 0 {
		return nil, code.GetError(code.UserDepartmentRelationNotExistError)
	}

	resp := &dtouser.UserDepartmentRelationDetailResp{
		UserDepartmentRelationID: userDepartmentRelationEntity.ID,
		TenantID:                 userDepartmentRelationEntity.TenantID,
		UserID:                   userDepartmentRelationEntity.UserID,
		DepartmentID:             userDepartmentRelationEntity.DepartmentID,
		IsPrimary:                userDepartmentRelationEntity.IsPrimary,
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: userDepartmentRelationEntity.CreatedAt.Unix(),
			UpdatedAt: userDepartmentRelationEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *userSvc) PageListUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationPageListReq) (*dtouser.UserDepartmentRelationPageListResp, error) {
	cond := &dao.UserDepartmentRelationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		IsPrimary:    req.IsPrimary,
	}
	userDepartmentRelationEntityList, total, err := dao.NewUserDepartmentRelationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageListUserDepartmentRelation] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationGetPageListError)
	}

	list := make([]dtouser.UserDepartmentRelationPageListItem, 0, len(userDepartmentRelationEntityList))
	for _, v := range userDepartmentRelationEntityList {
		list = append(list, dtouser.UserDepartmentRelationPageListItem{
			UserDepartmentRelationID: v.ID,
			TenantID:                 v.TenantID,
			UserID:                   v.UserID,
			DepartmentID:             v.DepartmentID,
			IsPrimary:                v.IsPrimary,
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserDepartmentRelationPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *userSvc) GetUserDepartmentRelationByUser(ctx *gin.Context, req *dtouser.UserDepartmentRelationByUserReq) (*dtouser.UserDepartmentRelationPageListResp, error) {
	cond := &dao.UserDepartmentRelationCond{
		UserID: req.UserID,
	}
	userDepartmentRelationEntityList, total, err := dao.NewUserDepartmentRelationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.GetUserDepartmentRelationByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationGetPageListError)
	}

	list := make([]dtouser.UserDepartmentRelationPageListItem, 0, len(userDepartmentRelationEntityList))
	for _, v := range userDepartmentRelationEntityList {
		list = append(list, dtouser.UserDepartmentRelationPageListItem{
			UserDepartmentRelationID: v.ID,
			TenantID:                 v.TenantID,
			UserID:                   v.UserID,
			DepartmentID:             v.DepartmentID,
			IsPrimary:                v.IsPrimary,
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserDepartmentRelationPageListResp{
		List:  list,
		Total: total,
	}, nil
}
