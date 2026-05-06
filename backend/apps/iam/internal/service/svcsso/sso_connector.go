package svcsso

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtosso"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objsso"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type SsoConnectorSvc interface {
	Create(ctx *gin.Context, req *dtosso.SsoConnectorCreateReq) (*dtosso.SsoConnectorCreateResp, error)
	Delete(ctx *gin.Context, req *dtosso.SsoConnectorDeleteReq) error
	Update(ctx *gin.Context, req *dtosso.SsoConnectorUpdateReq) error
	Detail(ctx *gin.Context, req *dtosso.SsoConnectorDetailReq) (*dtosso.SsoConnectorDetailResp, error)
	PageList(ctx *gin.Context, req *dtosso.SsoConnectorPageListReq) (*dtosso.SsoConnectorPageListResp, error)
}

type ssoConnectorSvc struct {
}

var _ SsoConnectorSvc = (*ssoConnectorSvc)(nil)

func NewSsoConnectorSvc() SsoConnectorSvc {
	return &ssoConnectorSvc{}
}

func (svc *ssoConnectorSvc) Create(ctx *gin.Context, req *dtosso.SsoConnectorCreateReq) (*dtosso.SsoConnectorCreateResp, error) {
	configJson, err := json.Marshal(req.Config)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Create] json.Marshal config fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SsoConnectorCreateError)
	}
	domainsJson, err := json.Marshal(req.Domains)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Create] json.Marshal domains fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SsoConnectorCreateError)
	}
	brandingJson, err := json.Marshal(req.Branding)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Create] json.Marshal branding fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SsoConnectorCreateError)
	}

	insertEntity := &model.SsoConnectorEntity{
		TenantID:           req.TenantID,
		ProviderName:       req.ProviderName,
		ConnectorName:      req.ConnectorName,
		Config:             configJson,
		Domains:            domainsJson,
		Branding:           brandingJson,
		SyncProfile:        req.SyncProfile,
		EnableTokenStorage: req.EnableTokenStorage,
		CreatedBy:          gincontext.GetUserID(ctx),
	}

	if err := dao.NewSsoConnectorDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcsso.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SsoConnectorCreateError)
	}
	return &dtosso.SsoConnectorCreateResp{
		SsoConnectorID: insertEntity.ID,
	}, nil
}

func (svc *ssoConnectorSvc) Delete(ctx *gin.Context, req *dtosso.SsoConnectorDeleteReq) error {
	ssoConnectorEntity, err := dao.NewSsoConnectorDao().GetByID(ctx, req.SsoConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SsoConnectorDeleteError)
	}
	if ssoConnectorEntity == nil || ssoConnectorEntity.ID == 0 {
		return code.GetError(code.SsoConnectorNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewSsoConnectorDao().Delete(ctx, req.SsoConnectorID, userID); err != nil {
		glog.Errorf(ctx, "[svcsso.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SsoConnectorDeleteError)
	}
	return nil
}

func (svc *ssoConnectorSvc) Update(ctx *gin.Context, req *dtosso.SsoConnectorUpdateReq) error {
	ssoConnectorEntity, err := dao.NewSsoConnectorDao().GetByID(ctx, req.SsoConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SsoConnectorUpdateError)
	}
	if ssoConnectorEntity == nil || ssoConnectorEntity.ID == 0 {
		return code.GetError(code.SsoConnectorNotExistError)
	}

	configJson, err := json.Marshal(req.Config)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Update] json.Marshal config fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SsoConnectorUpdateError)
	}
	domainsJson, err := json.Marshal(req.Domains)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Update] json.Marshal domains fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SsoConnectorUpdateError)
	}
	brandingJson, err := json.Marshal(req.Branding)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Update] json.Marshal branding fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SsoConnectorUpdateError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":            req.TenantID,
		"provider_name":         req.ProviderName,
		"connector_name":        req.ConnectorName,
		"config":                configJson,
		"domains":               domainsJson,
		"branding":              brandingJson,
		"sync_profile":          req.SyncProfile,
		"enable_token_storage":  req.EnableTokenStorage,
		"updated_by":            userID,
	}
	if err := dao.NewSsoConnectorDao().UpdateMap(ctx, req.SsoConnectorID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcsso.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SsoConnectorUpdateError)
	}
	return nil
}

func (svc *ssoConnectorSvc) Detail(ctx *gin.Context, req *dtosso.SsoConnectorDetailReq) (*dtosso.SsoConnectorDetailResp, error) {
	ssoConnectorEntity, err := dao.NewSsoConnectorDao().GetByID(ctx, req.SsoConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}
	if ssoConnectorEntity == nil || ssoConnectorEntity.ID == 0 {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	var config any
	if err := json.Unmarshal(ssoConnectorEntity.Config, &config); err != nil {
		glog.Errorf(ctx, "[svcsso.Detail] json.Unmarshal config fail, err:%v", err)
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}
	var domains any
	if err := json.Unmarshal(ssoConnectorEntity.Domains, &domains); err != nil {
		glog.Errorf(ctx, "[svcsso.Detail] json.Unmarshal domains fail, err:%v", err)
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}
	var branding any
	if err := json.Unmarshal(ssoConnectorEntity.Branding, &branding); err != nil {
		glog.Errorf(ctx, "[svcsso.Detail] json.Unmarshal branding fail, err:%v", err)
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}

	resp := &dtosso.SsoConnectorDetailResp{
		SsoConnectorID: ssoConnectorEntity.ID,
		SsoConnectorBaseInfo: objsso.SsoConnectorBaseInfo{
			TenantID:           ssoConnectorEntity.TenantID,
			ProviderName:       ssoConnectorEntity.ProviderName,
			ConnectorName:      ssoConnectorEntity.ConnectorName,
			Config:             config,
			Domains:            domains,
			Branding:           branding,
			SyncProfile:        ssoConnectorEntity.SyncProfile,
			EnableTokenStorage: ssoConnectorEntity.EnableTokenStorage,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: ssoConnectorEntity.CreatedAt.Unix(),
			UpdatedAt: ssoConnectorEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *ssoConnectorSvc) PageList(ctx *gin.Context, req *dtosso.SsoConnectorPageListReq) (*dtosso.SsoConnectorPageListResp, error) {
	cond := &dao.SsoConnectorCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:      req.TenantID,
		ProviderName:  req.ProviderName,
		ConnectorName: req.ConnectorName,
	}
	ssoConnectorEntityList, total, err := dao.NewSsoConnectorDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcsso.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SsoConnectorGetPageListError)
	}

	list := make([]dtosso.SsoConnectorPageListItem, 0, len(ssoConnectorEntityList))
	for _, v := range ssoConnectorEntityList {
		var config any
		if err := json.Unmarshal(v.Config, &config); err != nil {
			glog.Errorf(ctx, "[svcsso.PageList] json.Unmarshal config fail, err:%v", err)
			continue
		}
		var domains any
		if err := json.Unmarshal(v.Domains, &domains); err != nil {
			glog.Errorf(ctx, "[svcsso.PageList] json.Unmarshal domains fail, err:%v", err)
			continue
		}
		var branding any
		if err := json.Unmarshal(v.Branding, &branding); err != nil {
			glog.Errorf(ctx, "[svcsso.PageList] json.Unmarshal branding fail, err:%v", err)
			continue
		}
		list = append(list, dtosso.SsoConnectorPageListItem{
			SsoConnectorID: v.ID,
			SsoConnectorBaseInfo: objsso.SsoConnectorBaseInfo{
				TenantID:           v.TenantID,
				ProviderName:       v.ProviderName,
				ConnectorName:      v.ConnectorName,
				Config:             config,
				Domains:            domains,
				Branding:           branding,
				SyncProfile:        v.SyncProfile,
				EnableTokenStorage: v.EnableTokenStorage,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtosso.SsoConnectorPageListResp{
		List:  list,
		Total: total,
	}, nil
}