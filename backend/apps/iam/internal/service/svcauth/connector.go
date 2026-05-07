package svcauth

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoconnector"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ConnectorSvc interface {
	Create(ctx *gin.Context, req *dtoauth.ConnectorCreateReq) (*dtoauth.ConnectorCreateResp, error)
	Delete(ctx *gin.Context, req *dtoauth.ConnectorDeleteReq) error
	Update(ctx *gin.Context, req *dtoauth.ConnectorUpdateReq) error
	Detail(ctx *gin.Context, req *dtoauth.ConnectorDetailReq) (*dtoauth.ConnectorDetailResp, error)
	PageList(ctx *gin.Context, req *dtoauth.ConnectorPageListReq) (*dtoauth.ConnectorPageListResp, error)
	ListFactories(ctx *gin.Context) (*dtoconnector.ConnectorFactoryListResp, error)
	TestConnector(ctx *gin.Context, req *dtoconnector.ConnectorIDReq) (*dtoconnector.TestConnectorResp, error)
	GetAuthorizationUri(ctx *gin.Context, req *dtoconnector.AuthorizationUriReq) (*dtoconnector.AuthorizationUriResp, error)
}

type connectorSvc struct{}

var _ ConnectorSvc = (*connectorSvc)(nil)

func NewConnectorSvc() ConnectorSvc {
	return &connectorSvc{}
}

func (svc *connectorSvc) Create(ctx *gin.Context, req *dtoauth.ConnectorCreateReq) (*dtoauth.ConnectorCreateResp, error) {
	configJson, err := json.Marshal(req.Config)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.CreateConnector] json.Marshal config fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorCreateError)
	}
	metadataJson, err := json.Marshal(req.Metadata)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.CreateConnector] json.Marshal metadata fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorCreateError)
	}

	insertEntity := &model.ConnectorEntity{
		TenantID:           req.TenantID,
		SyncProfile:        req.SyncProfile,
		EnableTokenStorage: req.EnableTokenStorage,
		ConnectorID:        req.ConnectorID,
		Config:             configJson,
		Metadata:           metadataJson,
		CreatedBy:          gincontext.GetUserID(ctx),
	}

	if err := dao.NewConnectorDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.CreateConnector] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorCreateError)
	}
	return &dtoauth.ConnectorCreateResp{
		ConnectorID: insertEntity.ID,
	}, nil
}

func (svc *connectorSvc) Delete(ctx *gin.Context, req *dtoauth.ConnectorDeleteReq) error {
	connectorEntity, err := dao.NewConnectorDao().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.DeleteConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorDeleteError)
	}
	if connectorEntity == nil || connectorEntity.ID == 0 {
		return code.GetError(code.ConnectorNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewConnectorDao().Delete(ctx, req.ConnectorID, userID); err != nil {
		glog.Errorf(ctx, "[svcauth.DeleteConnector] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorDeleteError)
	}
	return nil
}

func (svc *connectorSvc) Update(ctx *gin.Context, req *dtoauth.ConnectorUpdateReq) error {
	connectorEntity, err := dao.NewConnectorDao().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.UpdateConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	if connectorEntity == nil || connectorEntity.ID == 0 {
		return code.GetError(code.ConnectorNotExistError)
	}

	configJson, err := json.Marshal(req.Config)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.UpdateConnector] json.Marshal config fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	metadataJson, err := json.Marshal(req.Metadata)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.UpdateConnector] json.Marshal metadata fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":            req.TenantID,
		"sync_profile":         req.SyncProfile,
		"enable_token_storage": req.EnableTokenStorage,
		"connector_id":         req.ConnectorID,
		"config":               configJson,
		"metadata":             metadataJson,
		"updated_by":           userID,
	}
	if err := dao.NewConnectorDao().UpdateMap(ctx, req.ConnectorID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcauth.UpdateConnector] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	return nil
}

func (svc *connectorSvc) Detail(ctx *gin.Context, req *dtoauth.ConnectorDetailReq) (*dtoauth.ConnectorDetailResp, error) {
	connectorEntity, err := dao.NewConnectorDao().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.DetailConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if connectorEntity == nil || connectorEntity.ID == 0 {
		return nil, code.GetError(code.ConnectorNotExistError)
	}

	var config any
	if err := json.Unmarshal(connectorEntity.Config, &config); err != nil {
		glog.Errorf(ctx, "[svcauth.DetailConnector] json.Unmarshal config fail, err:%v", err)
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	var metadata any
	if err := json.Unmarshal(connectorEntity.Metadata, &metadata); err != nil {
		glog.Errorf(ctx, "[svcauth.DetailConnector] json.Unmarshal metadata fail, err:%v", err)
		return nil, code.GetError(code.ConnectorGetDetailError)
	}

	resp := &dtoauth.ConnectorDetailResp{
		ConnectorID: connectorEntity.ID,
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:           connectorEntity.TenantID,
			SyncProfile:        connectorEntity.SyncProfile,
			EnableTokenStorage: connectorEntity.EnableTokenStorage,
			ConnectorID:        connectorEntity.ConnectorID,
			Config:             config,
			Metadata:           metadata,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: connectorEntity.CreatedAt.Unix(),
			UpdatedAt: connectorEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *connectorSvc) PageList(ctx *gin.Context, req *dtoauth.ConnectorPageListReq) (*dtoauth.ConnectorPageListResp, error) {
	cond := &dao.ConnectorCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    req.TenantID,
		ConnectorID: req.ConnectorID,
	}
	connectorEntityList, total, err := dao.NewConnectorDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.PageListConnector] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetPageListError)
	}

	list := make([]dtoauth.ConnectorPageListItem, 0, len(connectorEntityList))
	for _, v := range connectorEntityList {
		var config any
		if err := json.Unmarshal(v.Config, &config); err != nil {
			glog.Errorf(ctx, "[svcauth.PageListConnector] json.Unmarshal config fail, err:%v", err)
			continue
		}
		var metadata any
		if err := json.Unmarshal(v.Metadata, &metadata); err != nil {
			glog.Errorf(ctx, "[svcauth.PageListConnector] json.Unmarshal metadata fail, err:%v", err)
			continue
		}
		list = append(list, dtoauth.ConnectorPageListItem{
			ConnectorID: v.ID,
			ConnectorBaseInfo: objauth.ConnectorBaseInfo{
				TenantID:           v.TenantID,
				SyncProfile:        v.SyncProfile,
				EnableTokenStorage: v.EnableTokenStorage,
				ConnectorID:        v.ConnectorID,
				Config:             config,
				Metadata:           metadata,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtoauth.ConnectorPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *connectorSvc) ListFactories(ctx *gin.Context) (*dtoconnector.ConnectorFactoryListResp, error) {
	factories := []dtoconnector.ConnectorFactoryResp{
		{
			FactoryID:   "google",
			Name:        "google",
			DisplayName: "Google",
			Logo:        "https://cdn.anything.com/google-logo.png",
			IsStandard:  true,
		},
		{
			FactoryID:   "github",
			Name:        "github",
			DisplayName: "GitHub",
			Logo:        "https://cdn.anything.com/github-logo.png",
			IsStandard:  true,
		},
	}
	return &dtoconnector.ConnectorFactoryListResp{
		Factories: factories,
	}, nil
}

func (svc *connectorSvc) TestConnector(ctx *gin.Context, req *dtoconnector.ConnectorIDReq) (*dtoconnector.TestConnectorResp, error) {
	return &dtoconnector.TestConnectorResp{
		Success: true,
		Message: "连接成功",
	}, nil
}

func (svc *connectorSvc) GetAuthorizationUri(ctx *gin.Context, req *dtoconnector.AuthorizationUriReq) (*dtoconnector.AuthorizationUriResp, error) {
	authUri := "https://authorization.url/oauth/authorize?redirect_uri=" + req.RedirectURI + "&state=" + req.State
	return &dtoconnector.AuthorizationUriResp{
		AuthorizationUri: authUri,
	}, nil
}