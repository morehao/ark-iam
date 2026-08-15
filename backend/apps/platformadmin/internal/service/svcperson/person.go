package svcperson

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type PersonSvc interface {
	Create(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error)
	Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error
	Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error
	Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error)
	PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error)
	GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error)
}

type personSvc struct{}

var _ PersonSvc = (*personSvc)(nil)

func NewPersonSvc() PersonSvc {
	return &personSvc{}
}

func (svc *personSvc) Create(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error) {
	if err := ensurePersonIdentityVisibleToTenant(ctx, req.UserID); err != nil {
		return nil, err
	}
	detailJSON, err := json.Marshal(req.Detail)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.Create] json.Marshal detail fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityCreateError)
	}

	insertEntity := &model.UserIdentityEntity{
		PersonID:        req.UserID,
		Issuer:          req.Issuer,
		ExternalSubject: req.IdentityID,
		Detail:          detailJSON,
		CreatedBy:       gincontext.GetUserID(ctx),
	}
	if err := dao.NewUserIdentityDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcperson.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityCreateError)
	}
	return &dtouser.UserIdentityCreateResp{UserIdentityID: insertEntity.ID}, nil
}

func (svc *personSvc) Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error {
	entity, err := dao.NewUserIdentityDao().GetByID(ctx, req.UserIdentityID)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityDeleteError)
	}
	if entity == nil || entity.ID == "" {
		return code.GetError(code.UserIdentityNotExistError)
	}
	if err := ensurePersonIdentityVisibleToTenant(ctx, entity.PersonID); err != nil {
		return err
	}
	if err := dao.NewUserIdentityDao().Delete(ctx, req.UserIdentityID, gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[svcperson.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityDeleteError)
	}
	return nil
}

func (svc *personSvc) Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error {
	entity, err := dao.NewUserIdentityDao().GetByID(ctx, req.UserIdentityID)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityUpdateError)
	}
	if entity == nil || entity.ID == "" {
		return code.GetError(code.UserIdentityNotExistError)
	}
	if err := ensurePersonIdentityVisibleToTenant(ctx, entity.PersonID); err != nil {
		return err
	}
	if req.UserID != "" && req.UserID != entity.PersonID {
		if err := ensurePersonIdentityVisibleToTenant(ctx, req.UserID); err != nil {
			return err
		}
	}
	detailJSON, err := json.Marshal(req.Detail)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.Update] json.Marshal detail fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityUpdateError)
	}
	updateMap := map[string]any{
		"person_id":        req.UserID,
		"issuer":           req.Issuer,
		"external_subject": req.IdentityID,
		"detail":           detailJSON,
	}
	if operatorID := gincontext.GetUserID(ctx); operatorID != "" {
		updateMap["updated_by"] = operatorID
	}
	if err := dao.NewUserIdentityDao().UpdateMap(ctx, req.UserIdentityID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcperson.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityUpdateError)
	}
	return nil
}

func (svc *personSvc) Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error) {
	entity, err := dao.NewUserIdentityDao().GetByID(ctx, req.UserIdentityID)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.UserIdentityNotExistError)
	}
	if err := ensurePersonIdentityVisibleToTenant(ctx, entity.PersonID); err != nil {
		return nil, err
	}
	return buildPersonIdentityDetailResp(ctx, entity)
}

func (svc *personSvc) PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error) {
	personID := req.UserID
	if personID != "" {
		if err := ensurePersonIdentityVisibleToTenant(ctx, personID); err != nil {
			return nil, err
		}
	}
	cond := &dao.UserIdentityCond{
		BaseCond:        &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		PersonID:        personID,
		Issuer:          req.Issuer,
		ExternalSubject: req.IdentityID,
	}
	list, total, err := dao.NewUserIdentityDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcperson.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityGetPageListError)
	}
	return buildPersonIdentityPageListResp(ctx, list, total)
}

func (svc *personSvc) GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error) {
	if err := ensurePersonIdentityVisibleToTenant(ctx, req.UserID); err != nil {
		return nil, err
	}
	list, total, err := dao.NewUserIdentityDao().GetPageListByCond(ctx, &dao.UserIdentityCond{
		BaseCond: &gormdao.BaseCond{Page: 1, PageSize: 100},
		PersonID: req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcperson.GetByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityGetPageListError)
	}
	return buildPersonIdentityPageListResp(ctx, list, total)
}

func ensurePersonIdentityVisibleToTenant(ctx *gin.Context, personID string) error {
	if personID == "" {
		return code.GetError(code.UserIdentityNotExistError)
	}
	users, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{PersonID: personID})
	if err != nil {
		glog.Errorf(ctx, "[svcperson.ensurePersonIdentityVisibleToTenant] user dao GetListByCond fail, err:%v, personID:%s", err, personID)
		return code.GetError(code.UserIdentityGetDetailError)
	}
	tenantID := gincontext.GetTenantID(ctx)
	for _, userEntity := range users {
		if userEntity.TenantID == tenantID {
			return nil
		}
	}
	return code.GetError(code.UserIdentityNotExistError)
}

func buildPersonIdentityDetailResp(ctx *gin.Context, entity *model.UserIdentityEntity) (*dtouser.UserIdentityDetailResp, error) {
	var detail any
	if err := json.Unmarshal(entity.Detail, &detail); err != nil {
		glog.Errorf(ctx, "[svcperson.buildPersonIdentityDetailResp] json.Unmarshal detail fail, err:%v", err)
		return nil, code.GetError(code.UserIdentityGetDetailError)
	}
	return &dtouser.UserIdentityDetailResp{
		UserIdentityID: entity.ID,
		UserID:         entity.PersonID,
		Issuer:         entity.Issuer,
		IdentityID:     entity.ExternalSubject,
		Detail:         detail,
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: entity.CreatedAt.Unix(),
			UpdatedAt: entity.UpdatedAt.Unix(),
		},
	}, nil
}

func buildPersonIdentityPageListResp(ctx *gin.Context, entityList model.UserIdentityEntityList, total int64) (*dtouser.UserIdentityPageListResp, error) {
	list := make([]dtouser.UserIdentityPageListItem, 0, len(entityList))
	for _, v := range entityList {
		var detail any
		if err := json.Unmarshal(v.Detail, &detail); err != nil {
			glog.Errorf(ctx, "[svcperson.buildPersonIdentityPageListResp] json.Unmarshal detail fail, err:%v", err)
			continue
		}
		list = append(list, dtouser.UserIdentityPageListItem{
			UserIdentityID:   v.ID,
			UserID:           v.PersonID,
			Issuer:           v.Issuer,
			IdentityID:       v.ExternalSubject,
			Detail:           detail,
			OperatorBaseInfo: gobject.OperatorBaseInfo{UpdatedAt: v.UpdatedAt.Unix()},
		})
	}
	return &dtouser.UserIdentityPageListResp{List: list, Total: total}, nil
}
