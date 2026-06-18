package svcapplication

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationSvc interface {
	Create(ctx *gin.Context, req *dtoapplication.CreateReq) (*dtoapplication.CreateResp, error)
	Update(ctx *gin.Context, req *dtoapplication.UpdateReq) error
	Delete(ctx *gin.Context, req *dtoapplication.DeleteReq) error
	Detail(ctx *gin.Context, req *dtoapplication.DetailReq) (*dtoapplication.DetailResp, error)
	PageList(ctx *gin.Context, req *dtoapplication.PageListReq) (*dtoapplication.PageListResp, error)
}

type applicationSvc struct{}

var _ ApplicationSvc = (*applicationSvc)(nil)

func NewApplicationSvc() ApplicationSvc {
	return &applicationSvc{}
}

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoapplication.CreateReq) (*dtoapplication.CreateResp, error) {
	entity := &model.ApplicationEntity{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		HomepageURL: req.HomepageURL,
		Type:        req.Type,
		Visibility:  req.Visibility,
		Sort:        req.Sort,
		CreatedBy:   gincontext.GetUserID(ctx),
	}
	if err := dao.NewApplicationDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	return &dtoapplication.CreateResp{
		AppID: entity.ID,
		Code:     entity.Code,
	}, nil
}

func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoapplication.UpdateReq) error {
	updateMap := map[string]any{
		"name":         req.Name,
		"description":  req.Description,
		"logo_url":     req.LogoURL,
		"homepage_url": req.HomepageURL,
		"type":         req.Type,
		"visibility":   req.Visibility,
		"status":       req.Status,
		"sort":         req.Sort,
		"updated_by":   gincontext.GetUserID(ctx),
	}
	if err := dao.NewApplicationDao().UpdateMap(ctx, req.AppID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoapplication.DeleteReq) error {
	userID := gincontext.GetUserID(ctx)
	if err := dao.NewApplicationDao().Delete(ctx, req.AppID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoapplication.DetailReq) (*dtoapplication.DetailResp, error) {
	entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil || entity == nil || entity.ID == 0 {
		glog.Errorf(ctx, "[svcapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetDetailError)
	}
	return &dtoapplication.DetailResp{
		AppID:    entity.ID,
		Code:        entity.Code,
		Name:        entity.Name,
		Description: entity.Description,
		LogoURL:     entity.LogoURL,
		HomepageURL: entity.HomepageURL,
		Type:        entity.Type,
		Status:      entity.Status,
		Visibility:  entity.Visibility,
		Sort:        entity.Sort,
		CreatedAt:   entity.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoapplication.PageListReq) (*dtoapplication.PageListResp, error) {
	cond := &dao.ApplicationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		Name:   req.Name,
		Type:   req.Type,
		Status: req.Status,
	}
	list, total, err := dao.NewApplicationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetPageListError)
	}
	items := make([]dtoapplication.PageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtoapplication.PageListItem{
			AppID:    v.ID,
			Code:        v.Code,
			Name:        v.Name,
			Description: v.Description,
			Type:        v.Type,
			Status:      v.Status,
			Visibility:  v.Visibility,
			Sort:        v.Sort,
			CreatedAt:   v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dtoapplication.PageListResp{List: items, Total: total}, nil
}
