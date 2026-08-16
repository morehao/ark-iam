package svcauth

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoconnector"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/dbaccess/gormdao"
)

type fakeConnectorSSOSessionStore struct {
	createdFor string
}

var _ sso.SSOSessionStore = (*fakeConnectorSSOSessionStore)(nil)

func (f *fakeConnectorSSOSessionStore) CreateSession(ctx context.Context, personID string, amr []string) (string, error) {
	f.createdFor = personID
	return fmt.Sprintf("sso-session-%s", personID), nil
}

func (f *fakeConnectorSSOSessionStore) ValidateSession(ctx context.Context, sessionID string) (string, error) {
	return "", nil
}
func (f *fakeConnectorSSOSessionStore) SessionAMR(ctx context.Context, sessionID string) []string {
	return nil
}
func (f *fakeConnectorSSOSessionStore) SessionAuthTime(ctx context.Context, sessionID string) time.Time {
	return time.Now()
}
func (f *fakeConnectorSSOSessionStore) RevokeSession(ctx context.Context, sessionID string) error {
	return nil
}
func (f *fakeConnectorSSOSessionStore) RevokeSessionsByPersonID(ctx context.Context, personID string) error {
	return nil
}
func (f *fakeConnectorSSOSessionStore) HasActiveSession(ctx context.Context, personID string) (bool, error) {
	return false, nil
}

func TestConnectorCallbackReturnsPersonTokenWhenPersonHasMultipleTenants(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state-multi-tenant",
		Nonce:       "nonce-multi-tenant",
		ConnectorID: "11",
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiredAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	conn := &model.ConnectorEntity{
		Protocol:            connectorDriverTypeOAuth2,
		Provider:            connectorProviderGithub,
		Status:              connectorStatusEnabled,
		AllowAutoCreateUser: true,
		Config:              json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	conn.ID = "11"

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "21"}}, TenantID: "11", PersonID: "101", Name: "tenant-user-a"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "21"}}, TenantID: "11", PersonID: "101", Name: "tenant-user-a"},
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "22"}}, TenantID: "12", PersonID: "101", Name: "tenant-user-b"},
				}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getListByCondFunc: func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, error) {
				return model.TenantEntityList{
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "11"}}, Name: "tenant-a", Tag: "a"},
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "12"}}, Name: "tenant-b", Tag: "b"},
				}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{
			driverType: connectorDriverTypeOAuth2,
			exchangeCallbackFunc: func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
				return &ConnectorCallbackOutput{Identity: StandardIdentity{Issuer: "https://issuer.example.com", Subject: "sub-1", Email: "alice@example.com"}}, nil
			},
		}),
		connectorRepo: &fakeConnectorRuntimeRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) {
				return conn, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
				return &resolvedConnectorPerson{Person: &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}}, Username: model.StrPtr("alice")}}, nil
			},
		},
		ssoSessionStore: &fakeConnectorSSOSessionStore{},
		loginRecorder:   func(ctx *gin.Context, tenantID, userID string, success bool) {},
	}

	resp, err := svc.Callback(ginCtx, &dtoconnector.ConnectorCallbackReq{ConnectorID: "11", Code: "authorization-code", State: "callback-state-multi-tenant"})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resp == nil || resp.SSOSessionID == "" {
		t.Fatalf("expected sso session id in callback response, got %#v", resp)
	}
	if len(resp.Tenants) != 2 {
		t.Fatalf("expected 2 tenant options, got %#v", resp)
	}
	if resp.Tenants[0].TenantID != "11" || resp.Tenants[1].TenantID != "12" {
		t.Fatalf("expected tenant IDs [11 12], got %#v", resp.Tenants)
	}
}

func TestConnectorCallbackUsesIdentityResolverPath(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{State: "callback-state-resolver-path", Nonce: "nonce-resolver-path", ConnectorID: "19", RedirectURI: "https://app.example.com/oidc/callback", ExpiredAt: time.Now().Add(time.Minute)}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	conn := &model.ConnectorEntity{Protocol: connectorDriverTypeOAuth2, Provider: connectorProviderGithub, Status: connectorStatusEnabled, AllowAutoCreateUser: true, Config: json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`)}
	conn.ID = "19"
	conn.TenantID = "19"

	var insertedIdentity *model.UserIdentityEntity
	var insertedUser *model.UserEntity
	mapper := newIdentityMapper(
		&fakeConnectorPersonRepository{insertFunc: func(ctx context.Context, person *model.PersonEntity) error {
			person.ID = "501"
			return nil
		}},
		&fakeConnectorUserIdentityRepository{
			insertFunc: func(ctx context.Context, entity *model.UserIdentityEntity) error {
				clone := *entity
				insertedIdentity = &clone
				return nil
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
		// S4：自动创建用户需同时建立租户成员（UserEntity），注入 fake 断言
		WithIdentityMapperUserRepository(&fakeConnectorUserRepository{insertFunc: func(ctx context.Context, user *model.UserEntity) error {
			clone := *user
			insertedUser = &clone
			return nil
		}}),
	)

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "81"}}, TenantID: "31", PersonID: "501", Name: "tenant-user-a"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "81"}}, TenantID: "31", PersonID: "501", Name: "tenant-user-a"}}, nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{getListByCondFunc: func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, error) {
			return model.TenantEntityList{{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "31"}}, Name: "tenant-a", Tag: "a"}}, nil
		}}
	})
	defer restoreTenantStore()

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{driverType: connectorDriverTypeOAuth2, exchangeCallbackFunc: func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
			return &ConnectorCallbackOutput{Identity: StandardIdentity{Issuer: "https://issuer.example.com", Subject: "sub-resolver", Email: "resolver@example.com", DisplayName: "Resolver User"}}, nil
		}}),
		connectorRepo:    &fakeConnectorRuntimeRepository{getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) { return conn, nil }},
		stateStore:       stateStore,
		identityResolver: mapper,
		ssoSessionStore:  &fakeConnectorSSOSessionStore{},
		loginRecorder:    func(ctx *gin.Context, tenantID, userID string, success bool) {},
	}

	resp, err := svc.Callback(ginCtx, &dtoconnector.ConnectorCallbackReq{ConnectorID: "19", Code: "authorization-code", State: "callback-state-resolver-path"})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resp == nil || resp.SSOSessionID == "" {
		t.Fatalf("expected sso session response, got %#v", resp)
	}
	if insertedIdentity == nil {
		t.Fatal("expected identity resolver path to persist identity")
	}
	if insertedIdentity.PersonID != "501" || insertedIdentity.Issuer != "https://issuer.example.com" || insertedIdentity.ExternalSubject != "sub-resolver" {
		t.Fatalf("unexpected persisted identity: %#v", insertedIdentity)
	}
	// S4：自动创建用户必须同时建立租户成员（UserEntity）
	if insertedUser == nil {
		t.Fatal("expected auto-created user (tenant membership) to be inserted")
	}
	if insertedUser.PersonID != "501" || insertedUser.TenantID != "19" {
		t.Fatalf("unexpected persisted user: %#v", insertedUser)
	}
}
