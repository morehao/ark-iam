package svctenantapplication

import (
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenantapplication"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type TenantApplicationSvc interface {
	Create(ctx *gin.Context, req *dtotenantapplication.CreateReq) (*dtotenantapplication.CreateResp, error)
	Delete(ctx *gin.Context, req *dtotenantapplication.DeleteReq) error
	Update(ctx *gin.Context, req *dtotenantapplication.UpdateReq) error
	Detail(ctx *gin.Context, req *dtotenantapplication.DetailReq) (*dtotenantapplication.DetailResp, error)
	PageList(ctx *gin.Context, req *dtotenantapplication.PageListReq) (*dtotenantapplication.PageListResp, error)
}

type tenantApplicationSvc struct{}

var _ TenantApplicationSvc = (*tenantApplicationSvc)(nil)

func NewTenantApplicationSvc() TenantApplicationSvc {
	return &tenantApplicationSvc{}
}

func (svc *tenantApplicationSvc) Create(ctx *gin.Context, req *dtotenantapplication.CreateReq) (*dtotenantapplication.CreateResp, error) {
	entity := &model.TenantApplicationEntity{
		TenantID:  gincontext.GetTenantID(ctx),
		AppID:     req.AppID,
		Status:    req.Status,
		CreatedBy: gincontext.GetUserID(ctx),
	}
	if entity.Status == "" {
		entity.Status = model.AppStatusEnable
	}
	if req.Config != "" {
		entity.Config = datatypes.JSON([]byte(req.Config))
	}
	if err := dao.NewTenantApplicationDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	return &dtotenantapplication.CreateResp{TenantAppID: entity.ID}, nil
}

func (svc *tenantApplicationSvc) Delete(ctx *gin.Context, req *dtotenantapplication.DeleteReq) error {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil || entity == nil || entity.ID == 0 {
		return code.GetError(code.ApplicationNotExistError)
	}
	if entity.TenantID != gincontext.GetTenantID(ctx) {
		return code.GetError(code.ApplicationNotExistError)
	}
	if err := dao.NewTenantApplicationDao().Delete(ctx, req.TenantAppID, gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *tenantApplicationSvc) Update(ctx *gin.Context, req *dtotenantapplication.UpdateReq) error {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil || entity == nil || entity.ID == 0 {
		return code.GetError(code.ApplicationNotExistError)
	}
	if entity.TenantID != gincontext.GetTenantID(ctx) {
		return code.GetError(code.ApplicationNotExistError)
	}
	updateMap := map[string]any{
		"status":     req.Status,
		"updated_by": gincontext.GetUserID(ctx),
	}
	if req.Config != "" {
		updateMap["config"] = datatypes.JSON([]byte(req.Config))
	}
	if err := dao.NewTenantApplicationDao().UpdateMap(ctx, req.TenantAppID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *tenantApplicationSvc) Detail(ctx *gin.Context, req *dtotenantapplication.DetailReq) (*dtotenantapplication.DetailResp, error) {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil || entity == nil || entity.ID == 0 {
		return nil, code.GetError(code.ApplicationNotExistError)
	}
	if entity.TenantID != gincontext.GetTenantID(ctx) {
		return nil, code.GetError(code.ApplicationNotExistError)
	}
	return &dtotenantapplication.DetailResp{
		TenantAppID: entity.ID,
		TenantID:    entity.TenantID,
		AppID:       entity.AppID,
		Status:      entity.Status,
		Config:      string(entity.Config),
		CreatedAt:   entity.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (svc *tenantApplicationSvc) PageList(ctx *gin.Context, req *dtotenantapplication.PageListReq) (*dtotenantapplication.PageListResp, error) {
	cond := &dao.TenantApplicationCond{
		BaseCond: &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		TenantID: gincontext.GetTenantID(ctx),
		Status:   req.Status,
	}
	list, total, err := dao.NewTenantApplicationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetPageListError)
	}
	items := make([]dtotenantapplication.PageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtotenantapplication.PageListItem{
			TenantAppID: v.ID,
			TenantID:    v.TenantID,
			AppID:       v.AppID,
			Status:      v.Status,
			CreatedAt:   v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dtotenantapplication.PageListResp{List: items, Total: total}, nil
}
