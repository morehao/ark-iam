package svcscope

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtorole"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objscope"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ScopeSvc interface {
	Create(ctx *gin.Context, req *dtorole.ScopeCreateReq) (*dtorole.ScopeCreateResp, error)
	Delete(ctx *gin.Context, req *dtorole.ScopeDeleteReq) error
	Update(ctx *gin.Context, req *dtorole.ScopeUpdateReq) error
	Detail(ctx *gin.Context, req *dtorole.ScopeDetailReq) (*dtorole.ScopeDetailResp, error)
	PageList(ctx *gin.Context, req *dtorole.ScopePageListReq) (*dtorole.ScopePageListResp, error)
}

type scopeSvc struct {
}

var _ ScopeSvc = (*scopeSvc)(nil)

func NewScopeSvc() ScopeSvc {
	return &scopeSvc{}
}

func (svc *scopeSvc) Create(ctx *gin.Context, req *dtorole.ScopeCreateReq) (*dtorole.ScopeCreateResp, error) {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcscope.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceGetDetailError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return nil, code.GetError(code.ResourceNotExistError)
	}

	insertEntity := &model.ScopeEntity{
		TenantID:   req.TenantID,
		ResourceID: req.ResourceID,
		Name:       req.Name,
		Description: req.Description,
		CreatedBy:  gincontext.GetUserID(ctx),
	}

	if err := dao.NewScopeDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcscope.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeCreateError)
	}
	return &dtorole.ScopeCreateResp{
		ScopeID: insertEntity.ID,
	}, nil
}

func (svc *scopeSvc) Delete(ctx *gin.Context, req *dtorole.ScopeDeleteReq) error {
	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcscope.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeDeleteError)
	}
	if scopeEntity == nil || scopeEntity.ID == 0 {
		return code.GetError(code.ScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewScopeDao().Delete(ctx, req.ScopeID, userID); err != nil {
		glog.Errorf(ctx, "[svcscope.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeDeleteError)
	}
	return nil
}

func (svc *scopeSvc) Update(ctx *gin.Context, req *dtorole.ScopeUpdateReq) error {
	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcscope.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeUpdateError)
	}
	if scopeEntity == nil || scopeEntity.ID == 0 {
		return code.GetError(code.ScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":    req.TenantID,
		"resource_id":  req.ResourceID,
		"name":         req.Name,
		"description":  req.Description,
		"updated_by":   userID,
	}
	if err := dao.NewScopeDao().UpdateMap(ctx, req.ScopeID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcscope.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeUpdateError)
	}
	return nil
}

func (svc *scopeSvc) Detail(ctx *gin.Context, req *dtorole.ScopeDetailReq) (*dtorole.ScopeDetailResp, error) {
	scopeEntity, err := dao.NewScopeDao().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcscope.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeGetDetailError)
	}
	if scopeEntity == nil || scopeEntity.ID == 0 {
		return nil, code.GetError(code.ScopeNotExistError)
	}

	resp := &dtorole.ScopeDetailResp{
		ScopeID: scopeEntity.ID,
		ScopeBaseInfo: objscope.ScopeBaseInfo{
			TenantID:   scopeEntity.TenantID,
			ResourceID: scopeEntity.ResourceID,
			Name:       scopeEntity.Name,
			Description: scopeEntity.Description,
		},
	}
	return resp, nil
}

func (svc *scopeSvc) PageList(ctx *gin.Context, req *dtorole.ScopePageListReq) (*dtorole.ScopePageListResp, error) {
	cond := &dao.ScopeCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:   req.TenantID,
		ResourceID: req.ResourceID,
		Name:       req.Name,
	}
	scopeEntityList, total, err := dao.NewScopeDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcscope.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeGetPageListError)
	}

	list := make([]dtorole.ScopePageListItem, 0, len(scopeEntityList))
	for _, v := range scopeEntityList {
		list = append(list, dtorole.ScopePageListItem{
			ScopeID: v.ID,
			ScopeBaseInfo: objscope.ScopeBaseInfo{
				TenantID:   v.TenantID,
				ResourceID: v.ResourceID,
				Name:       v.Name,
				Description: v.Description,
			},
		})
	}
	return &dtorole.ScopePageListResp{
		List:  list,
		Total: total,
	}, nil
}