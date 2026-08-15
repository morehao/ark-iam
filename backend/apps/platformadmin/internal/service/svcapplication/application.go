package svcapplication

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/svcaudit"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationSvc interface {
	Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error)
	Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error
	Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error
	Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error)
	PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error)
}

type applicationSvc struct{}

var _ ApplicationSvc = (*applicationSvc)(nil)

func defaultTenantPolicy(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON("{}")
	}
	return datatypes.JSON(raw)
}

func NewApplicationSvc() ApplicationSvc {
	return &applicationSvc{}
}

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error) {
	entity := &model.ApplicationEntity{
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		LogoURL:      req.LogoURL,
		HomepageURL:  req.HomepageURL,
		Type:         req.Type,
		Visibility:   req.Visibility,
		TenantPolicy: defaultTenantPolicy(req.TenantPolicy),
		Sort:         req.Sort,
		CreatedBy:    gincontext.GetUserID(ctx),
	}
	if err := dao.NewApplicationDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionApplicationCreate,
		TenantID:   "",
		Result:     "success",
		TargetType: "application",
		TargetID:   entity.ID,
	})
	return &dtoapplication.ApplicationCreateResp{
		AppID: entity.ID,
		Code:  entity.Code,
	}, nil
}

func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error {
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
	if len(req.TenantPolicy) > 0 {
		updateMap["tenant_policy"] = datatypes.JSON(req.TenantPolicy)
	}
	if err := dao.NewApplicationDao().UpdateMap(ctx, req.AppID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error {
	entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	if entity != nil && entity.IsSystem {
		return code.GetError(code.ApplicationSystemBuiltInErr)
	}
	userID := gincontext.GetUserID(ctx)
	if err := dao.NewApplicationDao().Delete(ctx, req.AppID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error) {
	entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil || entity == nil || entity.ID == "" {
		glog.Errorf(ctx, "[svcapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetDetailError)
	}
	return &dtoapplication.ApplicationDetailResp{
		AppID:        entity.ID,
		Code:         entity.Code,
		Name:         entity.Name,
		Description:  entity.Description,
		LogoURL:      entity.LogoURL,
		HomepageURL:  entity.HomepageURL,
		Type:         entity.Type,
		Status:       entity.Status,
		Visibility:   entity.Visibility,
		Sort:         entity.Sort,
		TenantPolicy: json.RawMessage(entity.TenantPolicy),
		CreatedAt:    entity.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error) {
	cond := &dao.ApplicationCond{
		BaseCond: &gormdao.BaseCond{
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
			AppID:        v.ID,
			Code:         v.Code,
			Name:         v.Name,
			Description:  v.Description,
			Type:         v.Type,
			Status:       v.Status,
			Visibility:   v.Visibility,
			Sort:         v.Sort,
			TenantPolicy: json.RawMessage(v.TenantPolicy),
			CreatedAt:    v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dtoapplication.ApplicationPageListResp{List: items, Total: total}, nil
}
