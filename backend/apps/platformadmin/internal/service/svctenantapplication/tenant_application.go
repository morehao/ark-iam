package svctenantapplication

import (
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenantapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type TenantApplicationSvc interface {
	Create(ctx *gin.Context, req *dtotenantapplication.TenantApplicationCreateReq) (*dtotenantapplication.TenantApplicationCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDeleteReq) error
	Update(ctx *gin.Context, req *dtotenantapplication.TenantApplicationUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDetailReq) (*dtotenantapplication.TenantApplicationDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenantapplication.TenantApplicationPageListReq) (*dtotenantapplication.TenantApplicationPageListResp, error)
}

type tenantApplicationSvc struct{}

var _ TenantApplicationSvc = (*tenantApplicationSvc)(nil)

func NewTenantApplicationSvc() TenantApplicationSvc {
	return &tenantApplicationSvc{}
}

func (svc *tenantApplicationSvc) Create(ctx *gin.Context, req *dtotenantapplication.TenantApplicationCreateReq) (*dtotenantapplication.TenantApplicationCreateResp, error) {
	entity := &model.TenantApplicationEntity{
		TenantID:  gincontext.GetTenantIDString(ctx),
		AppID:     req.AppID,
		Status:    req.Status,
		CreatedBy: gincontext.GetUserIDString(ctx),
	}
	if entity.Status == "" {
		entity.Status = model.AppStatusEnable
	}
	// PG 下 not null JSON 列不接受 NULL：无配置时显式给默认值（与租户自建订阅路径一致）。
	if entity.Config == nil {
		entity.Config = datatypes.JSON("{}")
	}
	if entity.GrantedScope == nil {
		entity.GrantedScope = datatypes.JSON("[]")
	}
	if req.Config != "" {
		entity.Config = datatypes.JSON([]byte(req.Config))
	}
	if req.GrantedScope != "" {
		entity.GrantedScope = datatypes.JSON([]byte(req.GrantedScope))
	}
	if err := dao.NewTenantApplicationDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	return &dtotenantapplication.TenantApplicationCreateResp{TenantAppID: entity.ID}, nil
}

func (svc *tenantApplicationSvc) Delete(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDeleteReq) error {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil || entity == nil || entity.ID == "" {
		return code.GetError(code.ApplicationNotExistError)
	}
	if entity.TenantID != gincontext.GetTenantIDString(ctx) {
		return code.GetError(code.ApplicationNotExistError)
	}
	if err := dao.NewTenantApplicationDao().Delete(ctx, req.TenantAppID, gincontext.GetUserIDString(ctx)); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *tenantApplicationSvc) Update(ctx *gin.Context, req *dtotenantapplication.TenantApplicationUpdateReq) error {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil || entity == nil || entity.ID == "" {
		return code.GetError(code.ApplicationNotExistError)
	}
	if entity.TenantID != gincontext.GetTenantIDString(ctx) {
		return code.GetError(code.ApplicationNotExistError)
	}
	updateMap := map[string]any{
		"status":     req.Status,
		"updated_by": gincontext.GetUserIDString(ctx),
	}
	if req.Config != "" {
		updateMap["config"] = datatypes.JSON([]byte(req.Config))
	}
	if req.GrantedScope != "" {
		updateMap["granted_scope"] = datatypes.JSON([]byte(req.GrantedScope))
	}
	if err := dao.NewTenantApplicationDao().UpdateMap(ctx, req.TenantAppID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *tenantApplicationSvc) Detail(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDetailReq) (*dtotenantapplication.TenantApplicationDetailResp, error) {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil || entity == nil || entity.ID == "" {
		return nil, code.GetError(code.ApplicationNotExistError)
	}
	if entity.TenantID != gincontext.GetTenantIDString(ctx) {
		return nil, code.GetError(code.ApplicationNotExistError)
	}
	return &dtotenantapplication.TenantApplicationDetailResp{
		TenantAppID:  entity.ID,
		TenantID:     entity.TenantID,
		AppID:        entity.AppID,
		Status:       entity.Status,
		Config:       string(entity.Config),
		GrantedScope: string(entity.GrantedScope),
		CreatedAt:    entity.CreatedAt.Unix(),
	}, nil
}

func (svc *tenantApplicationSvc) PageList(ctx *gin.Context, req *dtotenantapplication.TenantApplicationPageListReq) (*dtotenantapplication.TenantApplicationPageListResp, error) {
	cond := &dao.TenantApplicationCond{
		BaseCond: &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		TenantID: gincontext.GetTenantIDString(ctx),
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
			CreatedAt:   v.CreatedAt.Unix(),
		})
	}
	return &dtotenantapplication.TenantApplicationPageListResp{List: items, Total: total}, nil
}
