package svcauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoconnector"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
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
	Authorize(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID string) (*dtoconnector.ConnectorAuthorizeResp, error)
	GetAuthorizationURL(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq) (*dtoconnector.ConnectorAuthorizeResp, error)
	Callback(ctx *gin.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error)
}

type connectorRuntimeRepository interface {
	GetByID(ctx context.Context, id string) (*model.ConnectorEntity, error)
}

type connectorIdentityResolver interface {
	Resolve(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error)
}

type connectorSvc struct {
	driverRegistry   *connectorDriverRegistry
	connectorRepo    connectorRuntimeRepository
	stateStore       ConnectorStateStore
	identityResolver connectorIdentityResolver
	ssoSessionStore  sso.SSOSessionStore
	tokenGenerator   func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error)
	loginRecorder    func(ctx *gin.Context, tenantID, userID string, success bool)
	stateGenerator   func() (string, error)
	nowFunc          func() time.Time
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

func connectorVisibleToTenant(entity *model.ConnectorEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
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

func (svc *connectorSvc) getSSOSessionStore() sso.SSOSessionStore {
	if svc.ssoSessionStore == nil {
		svc.ssoSessionStore = sso.NewSSOSessionStore()
	}
	return svc.ssoSessionStore
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

func buildConnectorInsertEntity(req *dtoauth.ConnectorCreateReq, createdBy string) (*model.ConnectorEntity, error) {
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

func buildConnectorUpdateMap(req *dtoauth.ConnectorUpdateReq, updatedBy string) (map[string]any, error) {
	configJson, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"tenant_id":              req.TenantID,
		"name":                   req.Name,
		"display_name":           req.DisplayName,
		"protocol":               req.Protocol,
		"provider":               req.Provider,
		"status":                 req.Status,
		"allow_auto_create_user": req.AllowAutoCreateUser,
		"allow_account_link":     req.AllowAccountLink,
		"sync_profile":           req.SyncProfile,
		"enable_token_storage":   req.EnableTokenStorage,
		"config":                 configJson,
		"claim_mapping":          mustMarshalJSON(req.ClaimMapping),
		"domain_policy":          mustMarshalJSON(req.DomainPolicy),
		"updated_by":             updatedBy,
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
	insertEntity, err := buildConnectorInsertEntity(req, gctx.GetUserID(ctx))
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
	connectorEntity, err := dao.NewConnectorDao().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.DeleteConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorDeleteError)
	}
	if !connectorVisibleToTenant(connectorEntity, gctx.GetTenantID(ctx)) {
		return code.GetError(code.ConnectorNotExistError)
	}

	userID := gctx.GetUserID(ctx)
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
	if !connectorVisibleToTenant(connectorEntity, gctx.GetTenantID(ctx)) {
		return code.GetError(code.ConnectorNotExistError)
	}

	updateMap, err := buildConnectorUpdateMap(req, gctx.GetUserID(ctx))
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
	connectorEntity, err := dao.NewConnectorDao().GetByID(ctx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.DetailConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if !connectorVisibleToTenant(connectorEntity, gctx.GetTenantID(ctx)) {
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
	connectorRepo := dao.NewConnectorDao()
	cond := &dao.ConnectorCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    gctx.GetTenantID(ctx),
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
				CreatedAt: v.CreatedAt.Unix(),
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
	connectorEntity, err := dao.NewConnectorDao().GetByID(ctx, req.ConnectorID)
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

func (svc *connectorSvc) Authorize(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID string) (*dtoconnector.ConnectorAuthorizeResp, error) {
	connectorEntity, err := svc.getConnectorRepo().GetByID(runtimeContext(ctx), connectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Authorize] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if connectorEntity == nil || connectorEntity.ID == "" || connectorEntity.Status != connectorStatusEnabled {
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
		ExpiredAt:   svc.getNowFunc()().Add(connectorStateTTL),
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
	if storedState == nil || storedState.ConnectorID == "" {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	if req.ConnectorID != "" && storedState.ConnectorID != req.ConnectorID {
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	connectorID := storedState.ConnectorID
	connectorEntity, err := svc.getConnectorRepo().GetByID(runtimeContext(ctx), connectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if connectorEntity == nil || connectorEntity.ID == "" || connectorEntity.Status != connectorStatusEnabled {
		return nil, code.GetError(code.ConnectorNotExistError)
	}
	driver, config, err := selectDriverForConnector(svc.getDriverRegistry(), connectorEntity)
	if err != nil {
		return nil, err
	}
	callbackOutput, err := driver.ExchangeCallback(ctx, &ConnectorCallbackInput{
		Config:      config,
		ConnectorID: connectorID,
		Code:        req.Code,
		State:       req.State,
		Nonce:       storedState.Nonce,
		RedirectURI: storedState.RedirectURI,
	})
	if err != nil {
		return nil, err
	}
	resolvedPerson, err := svc.getIdentityResolver().Resolve(runtimeContext(ctx), identityResolveInput{
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
	if resolvedPerson == nil || resolvedPerson.Person == nil || resolvedPerson.Person.ID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}
	if _, err := svc.getStateStore().Consume(runtimeContext(ctx), req.State); err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] consume state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}

	// Connector 登录成功后建立 SSO 会话，与账密/OIDC 登录走同一套 person 级中心会话；
	// 不再签发独立 HS256 person token（token 统一为 OIDC RS256，见设计文档 §4.4）。
	// amr 记录外部 IdP 认证方式（connector 名），供静默续登还原到 id_token。
	sessionID, err := svc.getSSOSessionStore().CreateSession(runtimeContext(ctx), resolvedPerson.Person.ID, []string{"ext"})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] create sso session fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}

	authRuntime := &authSvc{}
	tenantCtx := ctx
	if tenantCtx == nil {
		tenantCtx, _ = gin.CreateTestContext(nil)
		req, _ := http.NewRequest(http.MethodPost, "/", nil)
		tenantCtx.Request = req
	}
	_, tenants, err := authRuntime.listPersonTenants(tenantCtx, resolvedPerson.Person.ID)
	if err != nil {
		return nil, err
	}
	return &dtoauth.LoginResp{SSOSessionID: sessionID, Tenants: tenants}, nil
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
