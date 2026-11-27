package svcauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/morehao/ark-iam/apps/auth/dto/dtoauth"
	"github.com/morehao/ark-iam/apps/auth/dto/dtoconnector"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/connector"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/oidcctx"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"github.com/zitadel/oidc/v3/pkg/op"
)

const (
	connectorStateTTL = 10 * time.Minute
)

// ConnectorSvc 连接器「运行时」服务：仅承载登录链路（Authorize / Callback）。
// 连接器是平台级全局对象，管理面（CRUD/工厂/测试）已迁至 platformadmin/svcconnector；
// 领域不变式（驱动、校验、脱敏）下沉至 pkg/core/connector，auth 与 platformadmin 共用。
type ConnectorSvc interface {
	Authorize(ctx context.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID string) (*dtoconnector.ConnectorAuthorizeResp, error)
	Callback(ctx context.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error)
}

type connectorRuntimeRepository interface {
	GetByID(ctx context.Context, id string) (*model.ConnectorEntity, error)
}

type connectorIdentityResolver interface {
	Resolve(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error)
}

type connectorSvc struct {
	db               gormdao.DBGetter
	driverRegistry   *connector.DriverRegistry
	connectorRepo    connectorRuntimeRepository
	stateStore       ConnectorStateStore
	identityResolver connectorIdentityResolver
	authSvc          tenantProvider
	ssoSessionStore  sso.SSOSessionStore
	stateGenerator   func() (string, error)
	nowFunc          func() time.Time
}

// tenantProvider 抽象"查询自然人租户列表"能力，由 svcauth.AuthSvc 实现。
type tenantProvider interface {
	TenantsForPerson(ctx context.Context, personID string) ([]objauth.TenantOption, error)
}

var _ ConnectorSvc = (*connectorSvc)(nil)

func NewConnectorSvc(db gormdao.DBGetter) ConnectorSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &connectorSvc{
		db:             db,
		driverRegistry: connector.DefaultDriverRegistry(),
		authSvc:        NewAuthSvc(db),
	}
}

func (svc *connectorSvc) getAuthSvc() tenantProvider {
	if svc.authSvc == nil {
		svc.authSvc = NewAuthSvc(svc.db)
	}
	return svc.authSvc
}

func (svc *connectorSvc) getDriverRegistry() *connector.DriverRegistry {
	if svc.driverRegistry == nil {
		svc.driverRegistry = connector.DefaultDriverRegistry()
	}
	return svc.driverRegistry
}

func (svc *connectorSvc) getConnectorRepo() connectorRuntimeRepository {
	if svc.connectorRepo == nil {
		svc.connectorRepo = dao.NewConnectorDao(dao.WithDBGetter(svc.db))
	}
	return svc.connectorRepo
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

// Authorize 发起外部 IdP 授权：连接器为平台级全局对象、服务多租户，
// 运行时不再按「调用方租户 == 连接器租户」拦截（连接器无归属租户语义）；
// 仅校验连接器存在且启用。
func (svc *connectorSvc) Authorize(ctx context.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID string) (*dtoconnector.ConnectorAuthorizeResp, error) {
	connectorEntity, err := svc.getConnectorRepo().GetByID(runtimeContext(ctx), connectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Authorize] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if connectorEntity == nil || connectorEntity.ID == "" || connectorEntity.Status != model.ConnectorStatusEnable {
		return nil, code.GetError(code.ConnectorNotExistError)
	}
	// H12：redirect_uri 必须与本连接器配置的回调地址同源且为 https，
	// 防止把授权码导向攻击者控制的地址（开放重定向/授权码劫持）。
	if err := connector.ValidateConnectorRedirectURI(req.RedirectURI, connectorEntity); err != nil {
		glog.Warnf(ctx, "[svcauth.Authorize] invalid redirect uri, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	stateValue, err := svc.getStateGenerator()()
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Authorize] generate state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	driver, config, err := connector.SelectDriverForConnector(svc.getDriverRegistry(), connectorEntity)
	if err != nil {
		return nil, err
	}
	result, err := driver.BuildAuthorizationURL(ctx, &connector.AuthorizeInput{
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
		State:        stateValue,
		Nonce:        result.Nonce,
		ConnectorID:  connectorEntity.ID,
		TenantID:     connectorEntity.DefaultTenantID,
		RedirectURI:  req.RedirectURI,
		ExpiredAt:    svc.getNowFunc()().Add(connectorStateTTL),
		CodeVerifier: result.CodeVerifier,
	}); err != nil {
		glog.Errorf(ctx, "[svcauth.Authorize] save state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}
	return &dtoconnector.ConnectorAuthorizeResp{
		AuthorizationURL: result.AuthorizationURL,
	}, nil
}

func (svc *connectorSvc) Callback(ctx context.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error) {
	// 先原子消费 state（GetDel）：一次性使用，杜绝重放窗口；
	// 消费失败（已使用/过期/伪造）直接拒绝，不再继续 exchange。
	storedState, err := svc.getStateStore().Consume(runtimeContext(ctx), req.State)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] consume state fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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
	if connectorEntity == nil || connectorEntity.ID == "" || connectorEntity.Status != model.ConnectorStatusEnable {
		return nil, code.GetError(code.ConnectorNotExistError)
	}
	driver, config, err := connector.SelectDriverForConnector(svc.getDriverRegistry(), connectorEntity)
	if err != nil {
		return nil, err
	}
	callbackOutput, err := driver.ExchangeCallback(ctx, &connector.CallbackInput{
		Config:       config,
		ConnectorID:  connectorID,
		Code:         req.Code,
		State:        req.State,
		Nonce:        storedState.Nonce,
		CodeVerifier: storedState.CodeVerifier,
		RedirectURI:  storedState.RedirectURI,
	})
	if err != nil {
		return nil, err
	}

	allowAutoCreate := connectorEntity.AllowAutoCreateUser == model.ConnectorAutoCreateUserFlagEnable
	// 自动开户必须能确定落哪个租户（default_tenant_id）；为空则拒绝，避免建出无成员关系的悬空 person。
	if allowAutoCreate && connectorEntity.DefaultTenantID == "" {
		glog.Warnf(ctx, "[svcauth.Callback] allow_auto_create_user=enable but default_tenant_id empty, connectorID:%s", connectorID)
		return nil, code.GetError(code.ConnectorDefaultTenantRequiredError)
	}
	resolvedPerson, err := svc.getIdentityResolver().Resolve(runtimeContext(ctx), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  connectorEntity.ID,
			DefaultTenantID:     connectorEntity.DefaultTenantID,
			AllowAutoCreateUser: allowAutoCreate,
		},
		Identity: callbackOutput.Identity,
	})
	if err != nil {
		return nil, err
	}
	if resolvedPerson == nil || resolvedPerson.Person == nil || resolvedPerson.Person.ID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}

	// Connector 登录成功后建立 SSO 会话，与账密/OIDC 登录走同一套 person 级中心会话；
	// amr 记录外部 IdP 认证方式（connector 名），供静默续登还原到 id_token。
	ssCtx := withIssuerCtx(runtimeContext(ctx))
	sessionID, err := svc.getSSOSessionStore().CreateSession(ssCtx, resolvedPerson.Person.ID, []string{"ext"})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Callback] create sso session fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuthLoginFailedError)
	}

	tenants, err := svc.getAuthSvc().TenantsForPerson(ctx, resolvedPerson.Person.ID)
	if err != nil {
		return nil, err
	}
	return &dtoauth.LoginResp{SSOSessionID: sessionID, Tenants: tenants}, nil
}

func defaultConnectorStateGenerator() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// runtimeContext 保证下游拿到非 nil 的 context（context.Context 是接口，nil 会被
// glog / gorm 解引用 panic）。
func runtimeContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// withIssuerCtx 在 ctx 上注入本次请求 issuer（op.IssuerFromContext 优先，其次 pkg/oidcctx），
// 供 SSO 会话记录 issuer，用于跨 issuer 会话拒绝。
func withIssuerCtx(ctx context.Context) context.Context {
	iss := op.IssuerFromContext(ctx)
	if iss == "" {
		if v, ok := oidcctx.IssuerFromContext(ctx); ok {
			iss = v
		}
	}
	if iss != "" {
		ctx = context.WithValue(ctx, sso.ContextKeyIssuer, iss)
	}
	return ctx
}
