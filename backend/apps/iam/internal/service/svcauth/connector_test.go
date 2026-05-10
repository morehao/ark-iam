package svcauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoconnector"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/code"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeConnectorUserRepository struct {
	getByIDFunc func(ctx context.Context, id uint) (*model.UserEntity, error)
	insertFunc  func(ctx context.Context, user *model.UserEntity) error
}

func (f *fakeConnectorUserRepository) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeConnectorUserRepository) Insert(ctx context.Context, user *model.UserEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, user)
}

type fakeConnectorUserIdentityRepository struct {
	getByConnectorAndExternalSubjectFunc func(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error)
	insertFunc                           func(ctx context.Context, entity *model.UserIdentityEntity) error
	updateBindingFunc                    func(ctx context.Context, identityID, userID uint, issuer string, detail []byte) error
}

type fakeConnectorRuntimeRepository struct {
	getByIDFunc func(ctx context.Context, id uint) (*model.ConnectorEntity, error)
}

type fakeConnectorIdentityResolver struct {
	resolveFunc func(ctx context.Context, input identityResolveInput) (*model.UserEntity, error)
}

func (f *fakeConnectorIdentityResolver) Resolve(ctx context.Context, input identityResolveInput) (*model.UserEntity, error) {
	if f.resolveFunc == nil {
		return nil, nil
	}
	return f.resolveFunc(ctx, input)
}

func (f *fakeConnectorRuntimeRepository) GetByID(ctx context.Context, id uint) (*model.ConnectorEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

type fakeConnectorStateStore struct {
	saveFunc    func(ctx context.Context, state *ConnectorState) error
	loadFunc    func(ctx context.Context, state string) (*ConnectorState, error)
	consumeFunc func(ctx context.Context, state string) (*ConnectorState, error)
}

func (f *fakeConnectorStateStore) Save(ctx context.Context, state *ConnectorState) error {
	if f.saveFunc == nil {
		return nil
	}
	return f.saveFunc(ctx, state)
}

func (f *fakeConnectorStateStore) Load(ctx context.Context, state string) (*ConnectorState, error) {
	if f.loadFunc == nil {
		return nil, nil
	}
	return f.loadFunc(ctx, state)
}

func (f *fakeConnectorStateStore) Consume(ctx context.Context, state string) (*ConnectorState, error) {
	if f.consumeFunc == nil {
		return nil, nil
	}
	return f.consumeFunc(ctx, state)
}

type fakeConnectorDriver struct {
	driverType                 string
	validateConfigFunc         func(config ConnectorConfig) error
	buildAuthorizationURLFunc  func(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error)
	exchangeCallbackFunc       func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error)
	testConnectionFunc         func(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error)
}

func (f *fakeConnectorDriver) DriverType() string {
	return f.driverType
}

func (f *fakeConnectorDriver) ValidateConfig(config ConnectorConfig) error {
	if f.validateConfigFunc == nil {
		return nil
	}
	return f.validateConfigFunc(config)
}

func (f *fakeConnectorDriver) BuildAuthorizationURL(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error) {
	if f.buildAuthorizationURLFunc == nil {
		return nil, nil
	}
	return f.buildAuthorizationURLFunc(ctx, input)
}

func (f *fakeConnectorDriver) ExchangeCallback(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
	if f.exchangeCallbackFunc == nil {
		return nil, nil
	}
	return f.exchangeCallbackFunc(ctx, input)
}

func (f *fakeConnectorDriver) TestConnection(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error) {
	if f.testConnectionFunc == nil {
		return &ConnectorTestOutput{}, nil
	}
	return f.testConnectionFunc(ctx, input)
}

func (f *fakeConnectorUserIdentityRepository) GetByConnectorAndExternalSubject(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error) {
	if f.getByConnectorAndExternalSubjectFunc == nil {
		return nil, nil
	}
	return f.getByConnectorAndExternalSubjectFunc(ctx, tenantID, connectorID, externalSubject)
}

func (f *fakeConnectorUserIdentityRepository) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, entity)
}

func (f *fakeConnectorUserIdentityRepository) UpdateBinding(ctx context.Context, identityID, userID uint, issuer string, detail []byte) error {
	if f.updateBindingFunc == nil {
		return nil
	}
	return f.updateBindingFunc(ctx, identityID, userID, issuer, detail)
}

func TestBuildConnectorInsertEntityKeepsLegacyBusinessIdentifier(t *testing.T) {
	req := &dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:            1,
			Name:                "google-workspace",
			DisplayName:         "Google Workspace",
			Protocol:            "oidc",
			Provider:            "google",
			Status:              "enabled",
			AllowAutoCreateUser: 1,
			AllowAccountLink:    1,
			SyncProfile:         1,
			EnableTokenStorage:  1,
			Config: map[string]any{
				"clientId": "abc",
			},
			ClaimMapping: map[string]any{
				"email": "email",
			},
			DomainPolicy: map[string]any{
				"mode": "allow_all",
			},
		},
	}

	entity, err := buildConnectorInsertEntity(req, 99)
	if err != nil {
		t.Fatalf("buildConnectorInsertEntity returned error: %v", err)
	}
	if entity.Name != "google-workspace" {
		t.Fatalf("name should use new connector name, got: %q", entity.Name)
	}
	if entity.DisplayName != "Google Workspace" {
		t.Fatalf("display name should use new display name, got: %q", entity.DisplayName)
	}
	if entity.Name == "" {
		t.Fatalf("name should not be empty")
	}
}

func TestBuildConnectorInsertEntityDoesNotPersistLegacyMetadataAsClaimMapping(t *testing.T) {
	req := &dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:            1,
			Name:                "google-workspace",
			DisplayName:         "Google Workspace",
			Protocol:            "oidc",
			Provider:            "google",
			Status:              "enabled",
			AllowAutoCreateUser: 1,
			AllowAccountLink:    1,
			SyncProfile:         1,
			EnableTokenStorage:  1,
			Config: map[string]any{
				"clientId": "abc",
			},
			ClaimMapping: map[string]any{
				"email": "email",
			},
			DomainPolicy: map[string]any{
				"mode": "allow_all",
			},
		},
	}

	entity, err := buildConnectorInsertEntity(req, 99)
	if err != nil {
		t.Fatalf("buildConnectorInsertEntity returned error: %v", err)
	}
	if string(entity.ClaimMapping) != `{"email":"email"}` {
		t.Fatalf("claim mapping should persist new contract data, got: %s", string(entity.ClaimMapping))
	}
	if string(entity.DomainPolicy) != `{"mode":"allow_all"}` {
		t.Fatalf("domain policy should persist new contract data, got: %s", string(entity.DomainPolicy))
	}
}

func TestBuildConnectorUpdateMapDoesNotWritePrimaryKeyIntoNameFields(t *testing.T) {
	req := &dtoauth.ConnectorUpdateReq{
		ConnectorID: 42,
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:            1,
			Name:                "google-workspace",
			DisplayName:         "Google Workspace",
			Protocol:            "oidc",
			Provider:            "google",
			Status:              "enabled",
			AllowAutoCreateUser: 1,
			AllowAccountLink:    1,
			SyncProfile:         1,
			EnableTokenStorage:  1,
			Config: map[string]any{
				"clientId": "abc",
			},
			ClaimMapping: map[string]any{
				"email": "email",
			},
			DomainPolicy: map[string]any{
				"mode": "allow_all",
			},
		},
	}

	updateMap, err := buildConnectorUpdateMap(req, 100)
	if err != nil {
		t.Fatalf("buildConnectorUpdateMap returned error: %v", err)
	}
	if updateMap["name"] != "google-workspace" {
		t.Fatalf("update map should write name from new dto")
	}
	if updateMap["display_name"] != "Google Workspace" {
		t.Fatalf("update map should write display_name from new dto")
	}
	if string(updateMap["claim_mapping"].(json.RawMessage)) != `{"email":"email"}` {
		t.Fatalf("update map should persist claim_mapping from new dto")
	}
}

func TestIdentityMapperRejectsUnboundIdentityWhenAutoCreateDisabled(t *testing.T) {
	mapper := newIdentityMapper(
		&fakeConnectorUserRepository{},
		&fakeConnectorUserIdentityRepository{},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	_, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  11,
			TenantID:            22,
			AllowAutoCreateUser: false,
		},
		Identity: StandardIdentity{
			Issuer:  "https://issuer.example.com",
			Subject: "external-subject-1",
		},
	})
	if err == nil || err.Error() != code.GetError(code.UserNotExistError).Error() {
		t.Fatalf("expected user not exist error, got: %#v", err)
	}
}

func TestIdentityMapperReturnsBoundUser(t *testing.T) {
	expectedUser := &model.UserEntity{Model: model.UserEntity{}.Model, TenantID: 22, Username: "bound-user"}
	expectedUser.ID = 33

	mapper := newIdentityMapper(
		&fakeConnectorUserRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.UserEntity, error) {
				if id != expectedUser.ID {
					t.Fatalf("expected get by id %d, got %d", expectedUser.ID, id)
				}
				return expectedUser, nil
			},
		},
		&fakeConnectorUserIdentityRepository{
			getByConnectorAndExternalSubjectFunc: func(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error) {
				if tenantID != 22 || connectorID != 11 || externalSubject != "external-subject-1" {
					t.Fatalf("unexpected lookup input tenant=%d connector=%d subject=%q", tenantID, connectorID, externalSubject)
				}
				return &model.UserIdentityEntity{
					TenantID:         tenantID,
					UserID:           expectedUser.ID,
					ConnectorID:      connectorID,
					Issuer:           "https://issuer.example.com",
					ExternalSubject:  externalSubject,
				}, nil
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	user, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  11,
			TenantID:            22,
			AllowAutoCreateUser: true,
		},
		Identity: StandardIdentity{
			Issuer:  "https://issuer.example.com",
			Subject: "external-subject-1",
		},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if user != expectedUser {
		t.Fatalf("expected bound user pointer to be returned")
	}
}

func TestIdentityMapperAutoCreatesUserAndBindsIdentity(t *testing.T) {
	var insertedUser *model.UserEntity
	var insertedIdentity *model.UserIdentityEntity

	mapper := newIdentityMapper(
		&fakeConnectorUserRepository{
			insertFunc: func(ctx context.Context, user *model.UserEntity) error {
				insertedUser = user
				user.ID = 44
				return nil
			},
		},
		&fakeConnectorUserIdentityRepository{
			insertFunc: func(ctx context.Context, entity *model.UserIdentityEntity) error {
				insertedIdentity = entity
				return nil
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	user, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  11,
			TenantID:            22,
			AllowAutoCreateUser: true,
		},
		Identity: StandardIdentity{
			Issuer:      "https://issuer.example.com",
			Subject:     "external-subject-1",
			Email:       "user@example.com",
			Username:    "preferred-user",
			DisplayName: "Preferred User",
			AvatarURL:   "https://cdn.example.com/avatar.png",
			Claims: map[string]any{
				"department": "engineering",
			},
		},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if user == nil {
		t.Fatalf("expected created user")
	}
	if insertedUser == nil {
		t.Fatalf("expected user insert to be called")
	}
	if insertedUser.TenantID != 22 {
		t.Fatalf("expected tenant id 22, got %d", insertedUser.TenantID)
	}
	if insertedUser.Username != "preferred-user" {
		t.Fatalf("expected username from identity username, got %q", insertedUser.Username)
	}
	if insertedUser.PrimaryEmail != "user@example.com" {
		t.Fatalf("expected primary email from identity email, got %q", insertedUser.PrimaryEmail)
	}
	if insertedUser.Name != "Preferred User" {
		t.Fatalf("expected display name to map to user name, got %q", insertedUser.Name)
	}
	if insertedUser.Avatar != "https://cdn.example.com/avatar.png" {
		t.Fatalf("expected avatar url to map to user avatar, got %q", insertedUser.Avatar)
	}
	if insertedIdentity == nil {
		t.Fatalf("expected identity insert to be called")
	}
	if insertedIdentity.TenantID != 22 || insertedIdentity.UserID != 44 || insertedIdentity.ConnectorID != 11 {
		t.Fatalf("unexpected identity binding tenant=%d user=%d connector=%d", insertedIdentity.TenantID, insertedIdentity.UserID, insertedIdentity.ConnectorID)
	}
	if insertedIdentity.Issuer != "https://issuer.example.com" {
		t.Fatalf("expected issuer to be persisted, got %q", insertedIdentity.Issuer)
	}
	if insertedIdentity.ExternalSubject != "external-subject-1" {
		t.Fatalf("expected external subject to be persisted, got %q", insertedIdentity.ExternalSubject)
	}
	var detail StandardIdentity
	if err := json.Unmarshal(insertedIdentity.Detail, &detail); err != nil {
		t.Fatalf("unmarshal inserted identity detail failed: %v", err)
	}
	if !reflect.DeepEqual(detail, StandardIdentity{
		Issuer:      "https://issuer.example.com",
		Subject:     "external-subject-1",
		Email:       "user@example.com",
		Username:    "preferred-user",
		DisplayName: "Preferred User",
		AvatarURL:   "https://cdn.example.com/avatar.png",
		Claims: map[string]any{
			"department": "engineering",
		},
	}) {
		t.Fatalf("expected detail to persist standard identity, got %+v", detail)
	}
}

func TestIdentityMapperAutoCreateRollsBackUserWhenBindIdentityFails(t *testing.T) {
	ctx := newConnectorRuntimeContext(t)
	db := newConnectorIdentityMapperTestDB(t)
	mapper := newIdentityMapper(nil, nil, WithIdentityMapperDBProvider(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
	if err := db.Callback().Create().Before("gorm:create").Register("test_fail_user_identity_create", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Schema != nil && tx.Statement.Schema.Table == model.TableNameUserIdentity {
			tx.AddError(errors.New("bind identity failed"))
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	defer func() {
		_ = db.Callback().Create().Remove("test_fail_user_identity_create")
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	user, err := mapper.Resolve(ctx, identityResolveInput{
		Connector: ConnectorRuntime{ID: 11, TenantID: 22, AllowAutoCreateUser: true},
		Identity: StandardIdentity{
			Issuer:      "https://issuer.example.com",
			Subject:     "external-subject-rollback",
			Email:       "rollback@example.com",
			DisplayName: "Rollback User",
		},
	})
	if err == nil {
		t.Fatal("expected bind identity failure")
	}
	if user != nil {
		t.Fatalf("expected no user result on bind failure, got %+v", user)
	}

	var count int64
	if err := db.Model(&model.UserEntity{}).Where("tenant_id = ? AND primary_email = ?", 22, "rollback@example.com").Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove inserted user, got %d rows", count)
	}
}

func TestIdentityMapperAutoCreatePersistsUserAndIdentityWithinTransaction(t *testing.T) {
	ctx := newConnectorRuntimeContext(t)
	db := newConnectorIdentityMapperTestDB(t)
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	mapper := newIdentityMapper(nil, nil, WithIdentityMapperDBProvider(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	user, err := mapper.Resolve(ctx, identityResolveInput{
		Connector: ConnectorRuntime{ID: 11, TenantID: 22, AllowAutoCreateUser: true},
		Identity: StandardIdentity{
			Issuer:      "https://issuer.example.com",
			Subject:     "external-subject-success",
			Email:       "success@example.com",
			Username:    "success-user",
			DisplayName: "Success User",
		},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if user == nil || user.ID == 0 {
		t.Fatalf("expected created user, got %+v", user)
	}

	var users []model.UserEntity
	if err := db.Where("tenant_id = ? AND primary_email = ?", 22, "success@example.com").Find(&users).Error; err != nil {
		t.Fatalf("query users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 persisted user, got %d", len(users))
	}

	var identities []model.UserIdentityEntity
	if err := db.Where("tenant_id = ? AND connector_id = ? AND external_subject = ?", 22, 11, "external-subject-success").Find(&identities).Error; err != nil {
		t.Fatalf("query identities: %v", err)
	}
	if len(identities) != 1 {
		t.Fatalf("expected 1 persisted identity, got %d", len(identities))
	}
	if identities[0].UserID != users[0].ID || identities[0].UserID != user.ID {
		t.Fatalf("expected identity to bind created user, user=%d persisted=%d identity=%d", user.ID, users[0].ID, identities[0].UserID)
	}
}

func TestIdentityMapperPropagatesBoundUserLookupError(t *testing.T) {
	expectedErr := errors.New("user lookup failed")

	mapper := newIdentityMapper(
		&fakeConnectorUserRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.UserEntity, error) {
				return nil, expectedErr
			},
		},
		&fakeConnectorUserIdentityRepository{
			getByConnectorAndExternalSubjectFunc: func(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error) {
				return &model.UserIdentityEntity{Model: model.UserIdentityEntity{}.Model, UserID: 33}, nil
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	_, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{ID: 11, TenantID: 22, AllowAutoCreateUser: true},
		Identity:  StandardIdentity{Subject: "external-subject-1"},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected bound user lookup error to propagate, got %v", err)
	}
}

func TestIdentityMapperRepairsOrphanBindingInsteadOfReinserting(t *testing.T) {
	var inserted bool
	var updatedIdentityID uint
	var updatedUserID uint
	var updatedIssuer string
	var updatedDetail StandardIdentity

	mapper := newIdentityMapper(
		&fakeConnectorUserRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.UserEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, user *model.UserEntity) error {
				user.ID = 44
				return nil
			},
		},
		&fakeConnectorUserIdentityRepository{
			getByConnectorAndExternalSubjectFunc: func(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error) {
				entity := &model.UserIdentityEntity{UserID: 33, ConnectorID: connectorID, ExternalSubject: externalSubject, Issuer: "stale"}
				entity.ID = 77
				return entity, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserIdentityEntity) error {
				inserted = true
				return nil
			},
			updateBindingFunc: func(ctx context.Context, identityID, userID uint, issuer string, detail []byte) error {
				updatedIdentityID = identityID
				updatedUserID = userID
				updatedIssuer = issuer
				return json.Unmarshal(detail, &updatedDetail)
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	user, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{ID: 11, TenantID: 22, AllowAutoCreateUser: true},
		Identity: StandardIdentity{
			Issuer:      "https://issuer.example.com",
			Subject:     "external-subject-1",
			Email:       "user@example.com",
			DisplayName: "Preferred User",
		},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if user == nil || user.ID != 44 {
		t.Fatalf("expected repaired binding to return created user, got %+v", user)
	}
	if inserted {
		t.Fatal("expected orphan binding to be repaired via update, not insert")
	}
	if updatedIdentityID != 77 || updatedUserID != 44 {
		t.Fatalf("unexpected binding update target identity=%d user=%d", updatedIdentityID, updatedUserID)
	}
	if updatedIssuer != "https://issuer.example.com" {
		t.Fatalf("expected repaired binding issuer to be refreshed, got %q", updatedIssuer)
	}
	if updatedDetail.Subject != "external-subject-1" || updatedDetail.Email != "user@example.com" {
		t.Fatalf("expected repaired binding detail to persist latest identity, got %+v", updatedDetail)
	}
}

func TestConnectorServiceGetFactoryListReturnsTemplates(t *testing.T) {
	svc := &connectorSvc{}
	runtimeSvc, ok := any(svc).(interface {
		GetFactoryList(ctx *gin.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error)
	})
	if !ok {
		t.Fatal("connector service should expose GetFactoryList")
	}

	resp, err := runtimeSvc.GetFactoryList(nil, &dtoconnector.ConnectorFactoryListReq{})
	if err != nil {
		t.Fatalf("GetFactoryList returned error: %v", err)
	}
	if !reflect.DeepEqual(resp.List, defaultConnectorFactories()) {
		t.Fatalf("expected default connector factories, got %+v", resp.List)
	}
}

func TestConnectorServiceAuthorizeStoresStateAndReturnsAuthorizationURL(t *testing.T) {
	var savedState *ConnectorState
	var authorizeInput *ConnectorAuthorizeInput
	fixedNow := time.Unix(1710000000, 0)
	connectorEntity := &model.ConnectorEntity{
		TenantID:  22,
		Name:      "github-sso",
		Protocol:  connectorDriverTypeOAuth2,
		Provider:  connectorProviderGithub,
		Status:    connectorStatusEnabled,
		Config:    json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = 101

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{
			driverType: connectorDriverTypeOAuth2,
			buildAuthorizationURLFunc: func(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error) {
				authorizeInput = input
				return &ConnectorAuthorizeOutput{
					AuthorizationURL: "https://github.com/login/oauth/authorize?state=generated-state",
					Nonce:            "generated-nonce",
				}, nil
			},
		}),
		connectorRepo: &fakeConnectorRuntimeRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.ConnectorEntity, error) {
				if id != connectorEntity.ID {
					t.Fatalf("expected connector lookup by id %d, got %d", connectorEntity.ID, id)
				}
				return connectorEntity, nil
			},
		},
		stateStore: &fakeConnectorStateStore{
			saveFunc: func(ctx context.Context, state *ConnectorState) error {
				copyState := *state
				savedState = &copyState
				return nil
			},
		},
		stateGenerator: func() (string, error) {
			return "generated-state", nil
		},
		nowFunc: func() time.Time {
			return fixedNow
		},
	}
	runtimeSvc, ok := any(svc).(interface {
		Authorize(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID uint) (*dtoconnector.ConnectorAuthorizeResp, error)
	})
	if !ok {
		t.Fatal("connector service should expose Authorize")
	}

	resp, err := runtimeSvc.Authorize(nil, &dtoconnector.ConnectorAuthorizeReq{
		RedirectURI:  "https://app.example.com/oidc/callback",
		LoginHint:    "user@example.com",
		ResponseMode: "query",
	}, connectorEntity.ID)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if resp.AuthorizationURL != "https://github.com/login/oauth/authorize?state=generated-state" {
		t.Fatalf("unexpected authorization url: %q", resp.AuthorizationURL)
	}
	if authorizeInput == nil {
		t.Fatal("expected driver BuildAuthorizationURL to be called")
	}
	if authorizeInput.State != "generated-state" || authorizeInput.ConnectorID != connectorEntity.ID {
		t.Fatalf("unexpected authorize input: %+v", authorizeInput)
	}
	if authorizeInput.RedirectURI != "https://app.example.com/oidc/callback" {
		t.Fatalf("expected request redirect uri to be forwarded, got %q", authorizeInput.RedirectURI)
	}
	if savedState == nil {
		t.Fatal("expected connector state to be saved")
	}
	if savedState.State != "generated-state" || savedState.Nonce != "generated-nonce" {
		t.Fatalf("unexpected saved state payload: %+v", savedState)
	}
	if savedState.ConnectorID != connectorEntity.ID || savedState.TenantID != connectorEntity.TenantID {
		t.Fatalf("unexpected saved connector identity: %+v", savedState)
	}
	if savedState.RedirectURI != "https://app.example.com/oidc/callback" {
		t.Fatalf("expected saved redirect uri, got %q", savedState.RedirectURI)
	}
	if !savedState.ExpiresAt.Equal(fixedNow.Add(connectorStateTTL)) {
		t.Fatalf("expected expires at %v, got %v", fixedNow.Add(connectorStateTTL), savedState.ExpiresAt)
	}
}

func TestConnectorServiceCallbackConsumesStateAndInvokesDriver(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state",
		Nonce:       "nonce-1",
		ConnectorID: 101,
		TenantID:    22,
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	var callbackInput *ConnectorCallbackInput
	connectorEntity := &model.ConnectorEntity{
		TenantID:  22,
		Name:      "github-sso",
		Protocol:  connectorDriverTypeOAuth2,
		Provider:  connectorProviderGithub,
		Status:    connectorStatusEnabled,
		Config:    json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = 101

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{
			driverType: connectorDriverTypeOAuth2,
			exchangeCallbackFunc: func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
				callbackInput = input
				return &ConnectorCallbackOutput{
					Identity: StandardIdentity{Subject: "subject-1"},
				}, nil
			},
		}),
		connectorRepo: &fakeConnectorRuntimeRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.ConnectorEntity, error) {
				if id != connectorEntity.ID {
					t.Fatalf("expected connector lookup by id %d, got %d", connectorEntity.ID, id)
				}
				return connectorEntity, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*model.UserEntity, error) {
				user := &model.UserEntity{TenantID: 22, Username: "callback-user"}
				user.ID = 88
				return user, nil
			},
		},
		tokenGenerator: func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
			return &objauth.TokenInfo{AccessToken: "issued-access", RefreshToken: "issued-refresh", TokenType: "Bearer"}, nil
		},
		loginRecorder: func(ctx *gin.Context, tenantID, userID uint, success bool) {},
	}
	runtimeSvc, ok := any(svc).(interface {
		Callback(ctx *gin.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error)
	})
	if !ok {
		t.Fatal("connector service should expose Callback")
	}

	resp, err := runtimeSvc.Callback(nil, &dtoconnector.ConnectorCallbackReq{
		ConnectorID: 101,
		Code:        "authorization-code",
		State:       "callback-state",
	})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected placeholder login response")
	}
	if callbackInput == nil {
		t.Fatal("expected driver ExchangeCallback to be called")
	}
	if callbackInput.ConnectorID != 101 || callbackInput.Code != "authorization-code" || callbackInput.State != "callback-state" {
		t.Fatalf("unexpected callback input: %+v", callbackInput)
	}
	if callbackInput.Nonce != "nonce-1" {
		t.Fatalf("expected callback nonce from stored state, got %q", callbackInput.Nonce)
	}
	if callbackInput.RedirectURI != "https://app.example.com/oidc/callback" {
		t.Fatalf("expected callback redirect uri from stored state, got %q", callbackInput.RedirectURI)
	}
	if _, err := stateStore.Load(context.Background(), "callback-state"); !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("expected callback state to be consumed, got err=%v", err)
	}
}

func TestConnectorCallbackReturnsLoginResp(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state-login-resp",
		Nonce:       "nonce-login-resp",
		ConnectorID: 101,
		TenantID:    22,
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	connectorEntity := &model.ConnectorEntity{
		TenantID:            22,
		Protocol:            connectorDriverTypeOAuth2,
		Provider:            connectorProviderGithub,
		Status:              connectorStatusEnabled,
		AllowAutoCreateUser: 1,
		Config:              json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = 101

	resolvedUser := &model.UserEntity{TenantID: 22, Username: "connector-user"}
	resolvedUser.ID = 303

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{
			driverType: connectorDriverTypeOAuth2,
			exchangeCallbackFunc: func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
				return &ConnectorCallbackOutput{
					Identity:     StandardIdentity{Issuer: "https://issuer.example.com", Subject: "subject-1", Email: "connector@example.com"},
					AccessToken:  "provider-access-token",
					RefreshToken: "provider-refresh-token",
				}, nil
			},
		}),
		connectorRepo: &fakeConnectorRuntimeRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.ConnectorEntity, error) {
				return connectorEntity, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*model.UserEntity, error) {
				return resolvedUser, nil
			},
		},
		tokenGenerator: func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
			return &objauth.TokenInfo{
				AccessToken:  "iam-access-token",
				RefreshToken: "iam-refresh-token",
				ExpiresIn:    86400,
				TokenType:    "Bearer",
			}, nil
		},
		loginRecorder: func(ctx *gin.Context, tenantID, userID uint, success bool) {},
	}

	resp, err := svc.Callback(nil, &dtoconnector.ConnectorCallbackReq{ConnectorID: 101, Code: "authorization-code", State: "callback-state-login-resp"})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected login response")
	}
	if resp.AccessToken != "iam-access-token" || resp.RefreshToken != "iam-refresh-token" {
		t.Fatalf("expected callback to return issued iam tokens, got %+v", resp.TokenInfo)
	}
	if resp.AccessToken == "provider-access-token" || resp.RefreshToken == "provider-refresh-token" {
		t.Fatalf("expected callback to return iam tokens instead of provider tokens, got %+v", resp.TokenInfo)
	}
}

func TestConnectorCallbackInvokesIdentityResolverTokenGeneratorAndLoginRecorder(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state-runtime-hooks",
		Nonce:       "nonce-runtime-hooks",
		ConnectorID: 202,
		TenantID:    66,
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	connectorEntity := &model.ConnectorEntity{
		TenantID:            66,
		Protocol:            connectorDriverTypeOAuth2,
		Provider:            connectorProviderGithub,
		Status:              connectorStatusEnabled,
		AllowAutoCreateUser: 1,
		Config:              json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = 202

	resolvedUser := &model.UserEntity{TenantID: 66, Username: "mapped-user"}
	resolvedUser.ID = 909

	var resolvedInput identityResolveInput
	var tokenIssuedFor uint
	var recordTenantID uint
	var recordUserID uint
	var recordSuccess bool

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{
			driverType: connectorDriverTypeOAuth2,
			exchangeCallbackFunc: func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
				return &ConnectorCallbackOutput{
					Identity: StandardIdentity{
						Issuer:      "https://issuer.example.com",
						Subject:     "subject-202",
						Email:       "mapped@example.com",
						DisplayName: "Mapped User",
					},
				}, nil
			},
		}),
		connectorRepo: &fakeConnectorRuntimeRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.ConnectorEntity, error) {
				return connectorEntity, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*model.UserEntity, error) {
				resolvedInput = input
				return resolvedUser, nil
			},
		},
		tokenGenerator: func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
			tokenIssuedFor = userEntity.ID
			return &objauth.TokenInfo{AccessToken: "issued-access", RefreshToken: "issued-refresh", TokenType: "Bearer"}, nil
		},
		loginRecorder: func(ctx *gin.Context, tenantID, userID uint, success bool) {
			recordTenantID = tenantID
			recordUserID = userID
			recordSuccess = success
		},
	}

	_, err := svc.Callback(nil, &dtoconnector.ConnectorCallbackReq{ConnectorID: 202, Code: "authorization-code", State: "callback-state-runtime-hooks"})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resolvedInput.Connector.ID != 202 || resolvedInput.Connector.TenantID != 66 || !resolvedInput.Connector.AllowAutoCreateUser {
		t.Fatalf("unexpected connector runtime input: %+v", resolvedInput.Connector)
	}
	if resolvedInput.Identity.Subject != "subject-202" || resolvedInput.Identity.Email != "mapped@example.com" {
		t.Fatalf("unexpected standard identity input: %+v", resolvedInput.Identity)
	}
	if tokenIssuedFor != resolvedUser.ID {
		t.Fatalf("expected token generator to receive resolved user id %d, got %d", resolvedUser.ID, tokenIssuedFor)
	}
	if recordTenantID != 66 || recordUserID != resolvedUser.ID || !recordSuccess {
		t.Fatalf("unexpected login recorder input tenant=%d user=%d success=%v", recordTenantID, recordUserID, recordSuccess)
	}
	if _, err := stateStore.Load(context.Background(), "callback-state-runtime-hooks"); !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("expected callback state to be consumed after successful login flow, got err=%v", err)
	}
}

func TestConnectorServiceCallbackRetainsStateWhenDriverExchangeFails(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state-fail",
		Nonce:       "nonce-2",
		ConnectorID: 101,
		TenantID:    22,
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiresAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	expectedErr := errors.New("temporary exchange failure")
	conn := &model.ConnectorEntity{
		TenantID:  22,
		Protocol:  connectorDriverTypeOAuth2,
		Provider:  connectorProviderGithub,
		Status:    connectorStatusEnabled,
		Config:    json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	conn.ID = 101

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{
			driverType: connectorDriverTypeOAuth2,
			exchangeCallbackFunc: func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
				return nil, expectedErr
			},
		}),
		connectorRepo: &fakeConnectorRuntimeRepository{
			getByIDFunc: func(ctx context.Context, id uint) (*model.ConnectorEntity, error) {
				return conn, nil
			},
		},
		stateStore: stateStore,
	}

	_, err := svc.Callback(nil, &dtoconnector.ConnectorCallbackReq{ConnectorID: 101, Code: "authorization-code", State: "callback-state-fail"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected driver exchange error to propagate, got %v", err)
	}
	loaded, loadErr := stateStore.Load(context.Background(), "callback-state-fail")
	if loadErr != nil {
		t.Fatalf("expected state to remain for retry, got load err %v", loadErr)
	}
	if loaded == nil || loaded.State != "callback-state-fail" {
		t.Fatalf("expected retained state after driver failure, got %+v", loaded)
	}
}

func newConnectorIdentityMapperTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeConnectorTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserEntity{}, &model.UserIdentityEntity{}); err != nil {
		t.Fatalf("migrate user tables: %v", err)
	}
	return db
}

func newConnectorRuntimeContext(t *testing.T) context.Context {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req.Context()
}

func sanitizeConnectorTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}
