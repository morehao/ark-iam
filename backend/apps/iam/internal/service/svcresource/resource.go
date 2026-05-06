package svcresource

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtorole"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objresource"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ResourceSvc interface {
	Create(ctx *gin.Context, req *dtorole.ResourceCreateReq) (*dtorole.ResourceCreateResp, error)
	Delete(ctx *gin.Context, req *dtorole.ResourceDeleteReq) error
	Update(ctx *gin.Context, req *dtorole.ResourceUpdateReq) error
	Detail(ctx *gin.Context, req *dtorole.ResourceDetailReq) (*dtorole.ResourceDetailResp, error)
	PageList(ctx *gin.Context, req *dtorole.ResourcePageListReq) (*dtorole.ResourcePageListResp, error)
}

type resourceSvc struct {
}

var _ ResourceSvc = (*resourceSvc)(nil)

func NewResourceSvc() ResourceSvc {
	return &resourceSvc{}
}

func (svc *resourceSvc) Create(ctx *gin.Context, req *dtorole.ResourceCreateReq) (*dtorole.ResourceCreateResp, error) {
	insertEntity := &model.ResourceEntity{
		TenantID:       req.TenantID,
		Name:           req.Name,
		Indicator:      req.Indicator,
		IsDefault:      req.IsDefault,
		AccessTokenTtl: req.AccessTokenTtl,
		CreatedBy:      gincontext.GetUserID(ctx),
	}

	if err := dao.NewResourceDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcresource.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceCreateError)
	}
	return &dtorole.ResourceCreateResp{
		ResourceID: insertEntity.ID,
	}, nil
}

func (svc *resourceSvc) Delete(ctx *gin.Context, req *dtorole.ResourceDeleteReq) error {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcresource.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceDeleteError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return code.GetError(code.ResourceNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewResourceDao().Delete(ctx, req.ResourceID, userID); err != nil {
		glog.Errorf(ctx, "[svcresource.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceDeleteError)
	}
	return nil
}

func (svc *resourceSvc) Update(ctx *gin.Context, req *dtorole.ResourceUpdateReq) error {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcresource.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceUpdateError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return code.GetError(code.ResourceNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":        req.TenantID,
		"name":            req.Name,
		"indicator":       req.Indicator,
		"is_default":      req.IsDefault,
		"access_token_ttl": req.AccessTokenTtl,
		"updated_by":      userID,
	}
	if err := dao.NewResourceDao().UpdateMap(ctx, req.ResourceID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcresource.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ResourceUpdateError)
	}
	return nil
}

func (svc *resourceSvc) Detail(ctx *gin.Context, req *dtorole.ResourceDetailReq) (*dtorole.ResourceDetailResp, error) {
	resourceEntity, err := dao.NewResourceDao().GetByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcresource.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceGetDetailError)
	}
	if resourceEntity == nil || resourceEntity.ID == 0 {
		return nil, code.GetError(code.ResourceNotExistError)
	}

	resp := &dtorole.ResourceDetailResp{
		ResourceID: resourceEntity.ID,
		ResourceBaseInfo: objresource.ResourceBaseInfo{
			TenantID:       resourceEntity.TenantID,
			Name:           resourceEntity.Name,
			Indicator:      resourceEntity.Indicator,
			IsDefault:      resourceEntity.IsDefault,
			AccessTokenTtl: resourceEntity.AccessTokenTtl,
		},
	}
	return resp, nil
}

func (svc *resourceSvc) PageList(ctx *gin.Context, req *dtorole.ResourcePageListReq) (*dtorole.ResourcePageListResp, error) {
	cond := &dao.ResourceCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:  req.TenantID,
		Name:      req.Name,
		Indicator: req.Indicator,
	}
	resourceEntityList, total, err := dao.NewResourceDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcresource.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceGetPageListError)
	}

	list := make([]dtorole.ResourcePageListItem, 0, len(resourceEntityList))
	for _, v := range resourceEntityList {
		list = append(list, dtorole.ResourcePageListItem{
			ResourceID: v.ID,
			ResourceBaseInfo: objresource.ResourceBaseInfo{
				TenantID:       v.TenantID,
				Name:           v.Name,
				Indicator:      v.Indicator,
				IsDefault:      v.IsDefault,
				AccessTokenTtl: v.AccessTokenTtl,
			},
		})
	}
	return &dtorole.ResourcePageListResp{
		List:  list,
		Total: total,
	}, nil
}