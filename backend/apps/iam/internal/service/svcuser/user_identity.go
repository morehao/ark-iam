package svcuser

import (
	"encoding/json"

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
	detailJson, err := json.Marshal(req.Detail)
	if err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Create] json.Marshal detail fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityCreateError)
	}

	insertEntity := &model.UserIdentityEntity{
		TenantID:        req.TenantID,
		UserID:          req.UserID,
		Issuer:          req.Issuer,
		ExternalSubject: req.IdentityID,
		Detail:          detailJson,
		CreatedBy:       gincontext.GetUserID(ctx),
	}

	if err := dao.NewUserIdentityDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityCreateError)
	}
	return &dtouser.UserIdentityCreateResp{
		UserIdentityID: insertEntity.ID,
	}, nil
}

func (svc *userIdentitySvc) Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error {
	userIdentityEntity, err := dao.NewUserIdentityDao().GetByID(ctx, req.UserIdentityID)
	if err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityDeleteError)
	}
	if userIdentityEntity == nil || userIdentityEntity.ID == 0 {
		return code.GetError(code.UserIdentityNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewUserIdentityDao().Delete(ctx, req.UserIdentityID, userID); err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityDeleteError)
	}
	return nil
}

func (svc *userIdentitySvc) Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error {
	userIdentityEntity, err := dao.NewUserIdentityDao().GetByID(ctx, req.UserIdentityID)
	if err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityUpdateError)
	}
	if userIdentityEntity == nil || userIdentityEntity.ID == 0 {
		return code.GetError(code.UserIdentityNotExistError)
	}

	detailJson, err := json.Marshal(req.Detail)
	if err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Update] json.Marshal detail fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityUpdateError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":        req.TenantID,
		"user_id":          req.UserID,
		"issuer":           req.Issuer,
		"external_subject": req.IdentityID,
		"detail":           detailJson,
		"updated_by":       userID,
	}
	if err := dao.NewUserIdentityDao().UpdateMap(ctx, req.UserIdentityID, updateMap); err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityUpdateError)
	}
	return nil
}

func (svc *userIdentitySvc) Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error) {
	userIdentityEntity, err := dao.NewUserIdentityDao().GetByID(ctx, req.UserIdentityID)
	if err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityGetDetailError)
	}
	if userIdentityEntity == nil || userIdentityEntity.ID == 0 {
		return nil, code.GetError(code.UserIdentityNotExistError)
	}

	var detail any
	if err := json.Unmarshal(userIdentityEntity.Detail, &detail); err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.Detail] json.Unmarshal detail fail, err:%v", err)
		return nil, code.GetError(code.UserIdentityGetDetailError)
	}

	resp := &dtouser.UserIdentityDetailResp{
		UserIdentityID: userIdentityEntity.ID,
		TenantID:       userIdentityEntity.TenantID,
		UserID:         userIdentityEntity.UserID,
		Issuer:         userIdentityEntity.Issuer,
		IdentityID:     userIdentityEntity.ExternalSubject,
		Detail:         detail,
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: userIdentityEntity.CreatedAt.Unix(),
			UpdatedAt: userIdentityEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *userIdentitySvc) PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error) {
	cond := &dao.UserIdentityCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:        req.TenantID,
		ExternalSubject: req.IdentityID,
	}
	userIdentityEntityList, total, err := dao.NewUserIdentityDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityGetPageListError)
	}
	if req.UserID != 0 {
		filtered := make(model.UserIdentityEntityList, 0, len(userIdentityEntityList))
		for _, item := range userIdentityEntityList {
			if item.UserID == req.UserID {
				filtered = append(filtered, item)
			}
		}
		userIdentityEntityList = filtered
		total = int64(len(filtered))
	}

	list := make([]dtouser.UserIdentityPageListItem, 0, len(userIdentityEntityList))
	for _, v := range userIdentityEntityList {
		var detail any
		if err := json.Unmarshal(v.Detail, &detail); err != nil {
			glog.Errorf(ctx, "[userIdentitySvc.PageList] json.Unmarshal detail fail, err:%v", err)
			continue
		}
		list = append(list, dtouser.UserIdentityPageListItem{
			UserIdentityID: v.ID,
			TenantID:       v.TenantID,
			UserID:         v.UserID,
			Issuer:         v.Issuer,
			IdentityID:     v.ExternalSubject,
			Detail:         detail,
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserIdentityPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *userIdentitySvc) GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error) {
	cond := &dao.UserIdentityCond{}
	userIdentityEntityList, total, err := dao.NewUserIdentityDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[userIdentitySvc.GetByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityGetPageListError)
	}
	filtered := make(model.UserIdentityEntityList, 0, len(userIdentityEntityList))
	for _, item := range userIdentityEntityList {
		if item.UserID == req.UserID {
			filtered = append(filtered, item)
		}
	}
	userIdentityEntityList = filtered
	total = int64(len(filtered))

	list := make([]dtouser.UserIdentityPageListItem, 0, len(userIdentityEntityList))
	for _, v := range userIdentityEntityList {
		var detail any
		if err := json.Unmarshal(v.Detail, &detail); err != nil {
			glog.Errorf(ctx, "[userIdentitySvc.GetByUser] json.Unmarshal detail fail, err:%v", err)
			continue
		}
		list = append(list, dtouser.UserIdentityPageListItem{
			UserIdentityID: v.ID,
			TenantID:       v.TenantID,
			UserID:         v.UserID,
			Issuer:         v.Issuer,
			IdentityID:     v.ExternalSubject,
			Detail:         detail,
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserIdentityPageListResp{
		List:  list,
		Total: total,
	}, nil
}
