package svcappdefinition

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoappdefinition"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationSvc interface {
	Create(ctx *gin.Context, req *dtoappdefinition.CreateReq) (*dtoappdefinition.CreateResp, error)
	Update(ctx *gin.Context, req *dtoappdefinition.UpdateReq) error
	Delete(ctx *gin.Context, req *dtoappdefinition.DeleteReq) error
	Detail(ctx *gin.Context, req *dtoappdefinition.DetailReq) (*dtoappdefinition.DetailResp, error)
	PageList(ctx *gin.Context, req *dtoappdefinition.PageListReq) (*dtoappdefinition.PageListResp, error)
}

type applicationSvc struct{}

var _ ApplicationSvc = (*applicationSvc)(nil)

func NewApplicationSvc() ApplicationSvc {
	return &applicationSvc{}
}

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoappdefinition.CreateReq) (*dtoappdefinition.CreateResp, error) {
	entity := &model.ApplicationEntity{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		HomepageURL: req.HomepageURL,
		Type:        req.Type,
		Sort:        req.Sort,
		CreatedBy:   gincontext.GetUserID(ctx),
	}
	if err := dao.NewApplicationDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcappdefinition.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	return &dtoappdefinition.CreateResp{
		AppDefID: entity.ID,
		Code:     entity.Code,
	}, nil
}

func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoappdefinition.UpdateReq) error {
	updateMap := map[string]any{
		"name":         req.Name,
		"description":  req.Description,
		"logo_url":     req.LogoURL,
		"homepage_url": req.HomepageURL,
		"type":         req.Type,
		"status":       req.Status,
		"sort":         req.Sort,
		"updated_by":   gincontext.GetUserID(ctx),
	}
	if err := dao.NewApplicationDao().UpdateMap(ctx, req.AppDefID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcappdefinition.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoappdefinition.DeleteReq) error {
	userID := gincontext.GetUserID(ctx)
	if err := dao.NewApplicationDao().Delete(ctx, req.AppDefID, userID); err != nil {
		glog.Errorf(ctx, "[svcappdefinition.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoappdefinition.DetailReq) (*dtoappdefinition.DetailResp, error) {
	entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppDefID)
	if err != nil || entity == nil || entity.ID == 0 {
		glog.Errorf(ctx, "[svcappdefinition.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetDetailError)
	}
	return &dtoappdefinition.DetailResp{
		AppDefID:    entity.ID,
		Code:        entity.Code,
		Name:        entity.Name,
		Description: entity.Description,
		LogoURL:     entity.LogoURL,
		HomepageURL: entity.HomepageURL,
		Type:        entity.Type,
		Status:      entity.Status,
		Sort:        entity.Sort,
		CreatedAt:   entity.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoappdefinition.PageListReq) (*dtoappdefinition.PageListResp, error) {
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
		glog.Errorf(ctx, "[svcappdefinition.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetPageListError)
	}
	items := make([]dtoappdefinition.PageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtoappdefinition.PageListItem{
			AppDefID:    v.ID,
			Code:        v.Code,
			Name:        v.Name,
			Description: v.Description,
			Type:        v.Type,
			Status:      v.Status,
			Sort:        v.Sort,
			CreatedAt:   v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dtoappdefinition.PageListResp{List: items, Total: total}, nil
}
