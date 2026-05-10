package svcauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
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

const (
	connectorStatusEnabled = "enabled"
	connectorStateTTL      = 10 * time.Minute
)

type ConnectorSvc interface {
	Create(ctx *gin.Context, req *dtoauth.ConnectorCreateReq) (*dtoauth.ConnectorCreateResp, error)
	Delete(ctx *gin.Context, req *dtoauth.ConnectorDeleteReq) error
	Update(ctx *gin.Context, req *dtoauth.ConnectorUpdateReq) error
	Detail(ctx *gin.Context, req *dtoauth.ConnectorDetailReq) (*dtoauth.ConnectorDetailResp, error)
	PageList(ctx *gin.Context, req *dtoauth.ConnectorPageListReq) (*dtoauth.ConnectorPageListResp, error)
	GetFactoryList(ctx *gin.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error)
	ListFactories(ctx *gin.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error)
	TestConnector(ctx *gin.Context, req *dtoconnector.ConnectorIDReq) (*dtoconnector.TestConnectorResp, error)
	Authorize(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID uint) (*dtoconnector.ConnectorAuthorizeResp, error)
	GetAuthorizationURL(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq) (*dtoconnector.ConnectorAuthorizeResp, error)
	Callback(ctx *gin.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error)
}

type connectorRuntimeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.ConnectorEntity, error)
}

type connectorScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.ConnectorEntity, error)
	GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ConnectorEntityList, int64, error)
}

type connectorIdentityResolver interface {
	Resolve(ctx context.Context, input identityResolveInput) (*model.UserEntity, error)
}

type connectorSvc struct {
	driverRegistry *connectorDriverRegistry
	connectorRepo  connectorRuntimeRepository
	stateStore     ConnectorStateStore
	identityResolver connectorIdentityResolver
	tokenGenerator   func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error)
	loginRecorder    func(ctx *gin.Context, tenantID, userID uint, success bool)
	stateGenerator func() (string, error)
	nowFunc        func() time.Time
}

var _ ConnectorSvc = (*connectorSvc)(nil)

func NewConnectorSvc() ConnectorSvc {
	return &connectorSvc{driverRegistry: defaultConnectorDriverRegistry()}
}

func (svc *connectorSvc) getDriverRegistry() *connectorDriverRegistry {
	if svc.driverRegistry == nil {
		svc.driverRegistry = defaultConnectorDriverRegistry()
	}
	return svc.driverRegistry
}

func (svc *connectorSvc) getConnectorRepo() connectorRuntimeRepository {
	if svc.connectorRepo == nil {
		svc.connectorRepo = dao.NewConnectorDao()
	}
	return svc.connectorRepo
}

var newConnectorScopeRepo = func() connectorScopeRepository {
	return dao.NewConnectorDao()
}

func connectorVisibleToTenant(entity *model.ConnectorEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

func (svc *connectorSvc) getStateStore() ConnectorStateStore {
	if svc.stateStore == nil {
		svc.stateStore = NewRedisConnectorStateStore()
	}
	return svc.stateStore
}

func (svc *connectorSvc) getIdentityResolver() connectorIdentityResolver {
	if svc.identityResolver == nil {
		svc.identityResolver = newIdentityMapper(nil, nil)
	}
	return svc.identityResolver
}

func (svc *connectorSvc) getTokenGenerator() func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
	if svc.tokenGenerator == nil {
		authRuntime := &authSvc{jwtSecret: connectorJWTSignKey()}
		svc.tokenGenerator = authRuntime.generateToken
	}
	return svc.tokenGenerator
}

func (svc *connectorSvc) getLoginRecorder() func(ctx *gin.Context, tenantID, userID uint, success bool) {
	if svc.loginRecorder == nil {
		authRuntime := &authSvc{}
		svc.loginRecorder = authRuntime.recordLoginLog
	}
	return svc.loginRecorder
}

func (svc *connectorSvc) getStateGenerator() func() (string, error) {
	if svc.stateGenerator == nil {
		svc.stateGenerator = defaultConnectorStateGenerator
	}
	return svc.stateGenerator
}

func (svc *connectorSvc) getNowFunc() func() time.Time {
	if svc.nowFunc == nil {
		svc.nowFunc = time.Now
	}
	return svc.nowFunc
}

func buildConnectorInsertEntity(req *dtoauth.ConnectorCreateReq, createdBy uint) (*model.ConnectorEntity, error) {
	configJson, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}

	return &model.ConnectorEntity{
		TenantID:            req.TenantID,
		Name:                req.Name,
		DisplayName:         req.DisplayName,
		Protocol:            req.Protocol,
		Provider:            req.Provider,
		Status:              req.Status,
		AllowAutoCreateUser: req.AllowAutoCreateUser,
		AllowAccountLink:    req.AllowAccountLink,
		SyncProfile:         req.SyncProfile,
		EnableTokenStorage:  req.EnableTokenStorage,
		Config:              configJson,
		ClaimMapping:        mustMarshalJSON(req.ClaimMapping),
		DomainPolicy:        mustMarshalJSON(req.DomainPolicy),
		CreatedBy:           createdBy,
	}, nil
}

func buildConnectorUpdateMap(req *dtoauth.ConnectorUpdateReq, updatedBy uint) (map[string]any, error) {
	configJson, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"tenant_id":            req.TenantID,
		"name":                 req.Name,
		"display_name":         req.DisplayName,
		"protocol":             req.Protocol,
		"provider":             req.Provider,
		"status":               req.Status,
		"allow_auto_create_user": req.AllowAutoCreateUser,
		"allow_account_link":     req.AllowAccountLink,
		"sync_profile":         req.SyncProfile,
		"enable_token_storage": req.EnableTokenStorage,
		"config":               configJson,
		"claim_mapping":        mustMarshalJSON(req.ClaimMapping),
		"domain_policy":        mustMarshalJSON(req.DomainPolicy),
		"updated_by":           updatedBy,
	}, nil
}

func mustMarshalJSON(value any) json.RawMessage {
	if value == nil {
		return json.RawMessage("{}")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("{}")
	}
	if len(data) == 0 || string(data) == "null" {
		return json.RawMessage("{}")
	}
	return data
}

func (svc *connectorSvc) Create(ctx *gin.Context, req *dtoauth.ConnectorCreateReq) (*dtoauth.ConnectorCreateResp, error) {
	insertEntity, err := buildConnectorInsertEntity(req, gincontext.GetUserID(ctx))
	if err != nil {
		glog.Errorf(ctx, "[svcauth.CreateConnector] build insert entity fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorCreateError)
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
	connectorEntity, err := newConnectorScopeRepo().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.DeleteConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorDeleteError)
	}
	if !connectorVisibleToTenant(connectorEntity, gincontext.GetTenantID(ctx)) {
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
	connectorEntity, err := newConnectorScopeRepo().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.UpdateConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	if !connectorVisibleToTenant(connectorEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ConnectorNotExistError)
	}

	updateMap, err := buildConnectorUpdateMap(req, gincontext.GetUserID(ctx))
	if err != nil {
		glog.Errorf(ctx, "[svcauth.UpdateConnector] build update map fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	if err := dao.NewConnectorDao().UpdateMap(ctx, req.ConnectorID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcauth.UpdateConnector] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	return nil
}

func (svc *connectorSvc) Detail(ctx *gin.Context, req *dtoauth.ConnectorDetailReq) (*dtoauth.ConnectorDetailResp, error) {
	connectorEntity, err := newConnectorScopeRepo().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.DetailConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if !connectorVisibleToTenant(connectorEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.ConnectorNotExistError)
	}

	var config any
	if err := json.Unmarshal(connectorEntity.Config, &config); err != nil {
		glog.Errorf(ctx, "[svcauth.DetailConnector] json.Unmarshal config fail, err:%v", err)
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	resp := &dtoauth.ConnectorDetailResp{
		ConnectorID: connectorEntity.ID,
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:            connectorEntity.TenantID,
			Name:                connectorEntity.Name,
			DisplayName:         connectorEntity.DisplayName,
			Protocol:            connectorEntity.Protocol,
			Provider:            connectorEntity.Provider,
			Status:              connectorEntity.Status,
			AllowAutoCreateUser: connectorEntity.AllowAutoCreateUser,
			AllowAccountLink:    connectorEntity.AllowAccountLink,
			SyncProfile:         connectorEntity.SyncProfile,
			EnableTokenStorage:  connectorEntity.EnableTokenStorage,
			Config:              config,
			ClaimMapping:        unmarshalJSON(connectorEntity.ClaimMapping),
			DomainPolicy:        unmarshalJSON(connectorEntity.DomainPolicy),
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: connectorEntity.CreatedAt.Unix(),
			UpdatedAt: connectorEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *connectorSvc) PageList(ctx *gin.Context, req *dtoauth.ConnectorPageListReq) (*dtoauth.ConnectorPageListResp, error) {
	connectorRepo := newConnectorScopeRepo()
	cond := &dao.ConnectorCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    gincontext.GetTenantID(ctx),
		Protocol:    req.Protocol,
		Provider:    req.Provider,
		Status:      req.Status,
		Name:        req.Name,
		DisplayName: req.DisplayName,
	}
	connectorEntityList, total, err := connectorRepo.GetPageListByCond(ctx, cond)
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
		list = append(list, dtoauth.ConnectorPageListItem{
			ConnectorID: v.ID,
			ConnectorBaseInfo: objauth.ConnectorBaseInfo{
				TenantID:            v.TenantID,
				Name:                v.Name,
				DisplayName:         v.DisplayName,
				Protocol:            v.Protocol,
				Provider:            v.Provider,
				Status:              v.Status,
				AllowAutoCreateUser: v.AllowAutoCreateUser,
				AllowAccountLink:    v.AllowAccountLink,
				SyncProfile:         v.SyncProfile,
				EnableTokenStorage:  v.EnableTokenStorage,
				Config:              config,
				ClaimMapping:        unmarshalJSON(v.ClaimMapping),
				DomainPolicy:        unmarshalJSON(v.DomainPolicy),
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

func (svc *connectorSvc) GetFactoryList(ctx *gin.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error) {
	return &dtoconnector.ConnectorFactoryListResp{
		List: selectConnectorFactories(req, defaultConnectorFactories()),
	}, nil
}

func (svc *connectorSvc) ListFactories(ctx *gin.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error) {
	return svc.GetFactoryList(ctx, req)
}

func (svc *connectorSvc) TestConnector(ctx *gin.Context, req *dtoconnector.ConnectorIDReq) (*dtoconnector.TestConnectorResp, error) {
	connectorEntity, err := dao.NewConnectorDao().GetByID(ctx, uint(req.ConnectorID))
	if err != nil {
		glog.Errorf(ctx, "[svcauth.TestConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	driver, config, err := selectDriverForConnector(svc.getDriverRegistry(), connectorEntity)
	if err != nil {
		return nil, err
	}
	result, err := driver.TestConnection(ctx, &ConnectorTestInput{Config: config})
	if err != nil {
		return nil, err
	}
	return &dtoconnector.TestConnectorResp{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

func (svc *connectorSvc) Authorize(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID uint) (*dtoconnector.ConnectorAuthorizeResp, error) {
	connectorEntity, err := svc.getConnectorRepo().GetByID(runtimeContext(ctx), connectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Authorize] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if connectorEntity == nil || connectorEntity.ID == 0 || connectorEntity.Status != connectorStatusEnabled {
		return nil, code.GetError(code.ConnectorNotExistError)
	}
	stateValue, err := svc.getStateGenerator()()
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Authorize] generate state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	driver, config, err := selectDriverForConnector(svc.getDriverRegistry(), connectorEntity)
	if err != nil {
		return nil, err
	}
	result, err := driver.BuildAuthorizationURL(ctx, &ConnectorAuthorizeInput{
		Config:       config,
		ConnectorID:  connectorEntity.ID,
		RedirectURI:  req.RedirectURI,
		State:        stateValue,
		LoginHint:    req.LoginHint,
		ResponseMode: req.ResponseMode,
	})
	if err != nil {
		return nil, err
	}
	if err := svc.getStateStore().Save(runtimeContext(ctx), &ConnectorState{
		State:       stateValue,
		Nonce:       result.Nonce,
		ConnectorID: connectorEntity.ID,
		TenantID:    connectorEntity.TenantID,
		RedirectURI: req.RedirectURI,
		ExpiresAt:   svc.getNowFunc()().Add(connectorStateTTL),
	}); err != nil {
		glog.Errorf(ctx, "[svcauth.Authorize] save state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	return &dtoconnector.ConnectorAuthorizeResp{
		AuthorizationURL: result.AuthorizationURL,
	}, nil
}

func (svc *connectorSvc) GetAuthorizationURL(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq) (*dtoconnector.ConnectorAuthorizeResp, error) {
	return svc.Authorize(ctx, req, req.ConnectorID)
}

func (svc *connectorSvc) Callback(ctx *gin.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error) {
	storedState, err := svc.getStateStore().Load(runtimeContext(ctx), req.State)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] load state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	if storedState == nil || storedState.ConnectorID == 0 || storedState.ConnectorID != req.ConnectorID {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	connectorEntity, err := svc.getConnectorRepo().GetByID(runtimeContext(ctx), storedState.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if connectorEntity == nil || connectorEntity.ID == 0 || connectorEntity.Status != connectorStatusEnabled {
		return nil, code.GetError(code.ConnectorNotExistError)
	}
	driver, config, err := selectDriverForConnector(svc.getDriverRegistry(), connectorEntity)
	if err != nil {
		return nil, err
	}
	callbackOutput, err := driver.ExchangeCallback(ctx, &ConnectorCallbackInput{
		Config:      config,
		ConnectorID: connectorEntity.ID,
		Code:        req.Code,
		State:       req.State,
		Nonce:       storedState.Nonce,
		RedirectURI: storedState.RedirectURI,
	})
	if err != nil {
		return nil, err
	}
	userEntity, err := svc.getIdentityResolver().Resolve(runtimeContext(ctx), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  connectorEntity.ID,
			TenantID:            connectorEntity.TenantID,
			AllowAutoCreateUser: connectorEntity.AllowAutoCreateUser == 1,
		},
		Identity: callbackOutput.Identity,
	})
	if err != nil {
		return nil, err
	}
	if _, err := svc.getStateStore().Consume(runtimeContext(ctx), req.State); err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] consume state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	tokenInfo, err := svc.getTokenGenerator()(ctx, userEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] generate token fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TokenGenerateError)
	}
	svc.getLoginRecorder()(ctx, connectorEntity.TenantID, userEntity.ID, true)
	return &dtoauth.LoginResp{TokenInfo: *tokenInfo}, nil
}

func unmarshalJSON(data json.RawMessage) any {
	if len(data) == 0 {
		return map[string]any{}
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]any{}
	}
	if result == nil {
		return map[string]any{}
	}
	return result
}

func defaultConnectorStateGenerator() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func runtimeContext(ctx *gin.Context) context.Context {
	if ctx != nil && ctx.Request != nil {
		return ctx.Request.Context()
	}
	return context.Background()
}

func connectorJWTSignKey() string {
	if config.Conf == nil {
		return ""
	}
	return config.Conf.JWT.SignKey
}
