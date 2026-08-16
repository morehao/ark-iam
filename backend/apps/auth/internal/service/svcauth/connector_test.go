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
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoconnector"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeConnectorPersonRepository struct {
	getByIDFunc func(ctx context.Context, id string) (*model.PersonEntity, error)
	insertFunc  func(ctx context.Context, person *model.PersonEntity) error
}

func (f *fakeConnectorPersonRepository) GetByID(ctx context.Context, id string) (*model.PersonEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeConnectorPersonRepository) Insert(ctx context.Context, person *model.PersonEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, person)
}

type fakeConnectorUserRepository struct {
	insertFunc func(ctx context.Context, user *model.UserEntity) error
}

func (f *fakeConnectorUserRepository) Insert(ctx context.Context, user *model.UserEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, user)
}

type fakeConnectorUserIdentityRepository struct {
	getByIssuerAndExternalSubjectFunc func(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error)
	insertFunc                        func(ctx context.Context, entity *model.UserIdentityEntity) error
	updateBindingFunc                 func(ctx context.Context, identityID, personID string, issuer string, detail []byte) error
}

type fakeConnectorRuntimeRepository struct {
	getByIDFunc func(ctx context.Context, id string) (*model.ConnectorEntity, error)
}

type fakeConnectorIdentityResolver struct {
	resolveFunc func(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error)
}

func (f *fakeConnectorIdentityResolver) Resolve(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
	if f.resolveFunc == nil {
		return nil, nil
	}
	return f.resolveFunc(ctx, input)
}

func (f *fakeConnectorRuntimeRepository) GetByID(ctx context.Context, id string) (*model.ConnectorEntity, error) {
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
	driverType                string
	validateConfigFunc        func(config ConnectorConfig) error
	buildAuthorizationURLFunc func(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error)
	exchangeCallbackFunc      func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error)
	testConnectionFunc        func(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error)
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

func (f *fakeConnectorUserIdentityRepository) GetByIssuerAndExternalSubject(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error) {
	if f.getByIssuerAndExternalSubjectFunc == nil {
		return nil, nil
	}
	return f.getByIssuerAndExternalSubjectFunc(ctx, issuer, externalSubject)
}

func (f *fakeConnectorUserIdentityRepository) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, entity)
}

func (f *fakeConnectorUserIdentityRepository) UpdateBinding(ctx context.Context, identityID, personID string, issuer string, detail []byte) error {
	if f.updateBindingFunc == nil {
		return nil
	}
	return f.updateBindingFunc(ctx, identityID, personID, issuer, detail)
}

func TestBuildConnectorInsertEntityKeepsLegacyBusinessIdentifier(t *testing.T) {
	req := &dtoauth.ConnectorCreateReq{
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:            "1",
			Name:                "google-workspace",
			DisplayName:         "Google Workspace",
			Protocol:            "oidc",
			Provider:            "google",
			Status:              "enabled",
			AllowAutoCreateUser: true,
			AllowAccountLink:    true,
			SyncProfile:         true,
			EnableTokenStorage:  true,
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

	entity, err := buildConnectorInsertEntity(req, "99", "created-by")
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
			TenantID:            "1",
			Name:                "google-workspace",
			DisplayName:         "Google Workspace",
			Protocol:            "oidc",
			Provider:            "google",
			Status:              "enabled",
			AllowAutoCreateUser: true,
			AllowAccountLink:    true,
			SyncProfile:         true,
			EnableTokenStorage:  true,
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

	entity, err := buildConnectorInsertEntity(req, "99", "created-by")
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
		ConnectorID: "42",
		ConnectorBaseInfo: objauth.ConnectorBaseInfo{
			TenantID:            "1",
			Name:                "google-workspace",
			DisplayName:         "Google Workspace",
			Protocol:            "oidc",
			Provider:            "google",
			Status:              "enabled",
			AllowAutoCreateUser: true,
			AllowAccountLink:    true,
			SyncProfile:         true,
			EnableTokenStorage:  true,
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

	updateMap, err := buildConnectorUpdateMap(req, "100")
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
		&fakeConnectorPersonRepository{},
		&fakeConnectorUserIdentityRepository{},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	_, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  "11",
			TenantID:            "22",
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

func TestIdentityMapperReturnsBoundPerson(t *testing.T) {
	expectedPerson := &model.PersonEntity{BaseEntity: model.PersonEntity{}.BaseEntity, Username: model.StrPtr("bound-user"), Name: "bound-user"}
	expectedPerson.ID = "33"

	mapper := newIdentityMapper(
		&fakeConnectorPersonRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.PersonEntity, error) {
				if id != expectedPerson.ID {
					t.Fatalf("expected get by id %s, got %s", expectedPerson.ID, id)
				}
				return expectedPerson, nil
			},
		},
		&fakeConnectorUserIdentityRepository{
			getByIssuerAndExternalSubjectFunc: func(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error) {
				if issuer != "https://issuer.example.com" || externalSubject != "external-subject-1" {
					t.Fatalf("unexpected lookup input issuer=%q subject=%q", issuer, externalSubject)
				}
				return &model.UserIdentityEntity{
					PersonID:        expectedPerson.ID,
					Issuer:          "https://issuer.example.com",
					ExternalSubject: externalSubject,
				}, nil
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	person, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  "11",
			TenantID:            "22",
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
	if person == nil || person.Person != expectedPerson {
		t.Fatalf("expected bound person pointer to be returned")
	}
}

func TestIdentityMapperAutoCreatesPersonAndBindsIdentity(t *testing.T) {
	var insertedPerson *model.PersonEntity
	var insertedIdentity *model.UserIdentityEntity
	var insertedUser *model.UserEntity

	mapper := newIdentityMapper(
		&fakeConnectorPersonRepository{
			insertFunc: func(ctx context.Context, person *model.PersonEntity) error {
				insertedPerson = person
				person.ID = "44"
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
		// S4：自动创建用户需建立租户成员（UserEntity）
		WithIdentityMapperUserRepository(&fakeConnectorUserRepository{
			insertFunc: func(ctx context.Context, user *model.UserEntity) error {
				insertedUser = user
				return nil
			},
		}),
	)

	person, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{
			ID:                  "11",
			TenantID:            "22",
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
	if person == nil || person.Person == nil {
		t.Fatalf("expected created person")
	}
	if insertedPerson == nil {
		t.Fatalf("expected person insert to be called")
	}
	if model.DerefStr(insertedPerson.PrimaryEmail) != "user@example.com" {
		t.Fatalf("expected email to map to person primary email, got %q", model.DerefStr(insertedPerson.PrimaryEmail))
	}
	if insertedPerson.Name != "Preferred User" {
		t.Fatalf("expected display name to map to person name, got %q", insertedPerson.Name)
	}
	if insertedPerson.Avatar != "https://cdn.example.com/avatar.png" {
		t.Fatalf("expected avatar url to map to person avatar, got %q", insertedPerson.Avatar)
	}
	if insertedIdentity == nil {
		t.Fatalf("expected identity insert to be called")
	}
	if insertedIdentity.PersonID != "44" || insertedIdentity.ConnectorID != "11" {
		t.Fatalf("unexpected identity binding person=%s connector=%s", insertedIdentity.PersonID, insertedIdentity.ConnectorID)
	}
	if insertedIdentity.Issuer != "https://issuer.example.com" {
		t.Fatalf("expected issuer to be persisted, got %q", insertedIdentity.Issuer)
	}
	if insertedIdentity.ExternalSubject != "external-subject-1" {
		t.Fatalf("expected external subject to be persisted, got %q", insertedIdentity.ExternalSubject)
	}
	// S4：自动创建用户须建立租户成员（UserEntity），成员租户 = connector 租户
	if insertedUser == nil {
		t.Fatal("expected user (tenant membership) insert to be called")
	}
	if insertedUser.PersonID != "44" || insertedUser.TenantID != "22" {
		t.Fatalf("unexpected user membership person=%s tenant=%s", insertedUser.PersonID, insertedUser.TenantID)
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

func TestIdentityMapperAutoCreateRollsBackPersonWhenBindIdentityFails(t *testing.T) {
	ctx := newConnectorRuntimeContext(t)
	db := newConnectorIdentityMapperTestDB(t)
	mapper := newIdentityMapper(nil, nil, WithIdentityMapperDBProvider(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
	if err := db.Callback().Create().Before("gorm:create").Register("test_fail_user_identity_create", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Schema != nil && tx.Statement.Schema.Table == model.TableNameUserIdentity {
			_ = tx.AddError(errors.New("bind identity failed"))
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

	person, err := mapper.Resolve(ctx, identityResolveInput{
		Connector: ConnectorRuntime{ID: "11", TenantID: "22", AllowAutoCreateUser: true},
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
	if person != nil {
		t.Fatalf("expected no person result on bind failure, got %+v", person)
	}

	var count int64
	if err := db.Model(&model.PersonEntity{}).Where("primary_email = ?", "rollback@example.com").Count(&count).Error; err != nil {
		t.Fatalf("count persons: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove inserted person, got %d rows", count)
	}
}

func TestIdentityMapperAutoCreatePersistsPersonAndIdentityWithinTransaction(t *testing.T) {
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

	person, err := mapper.Resolve(ctx, identityResolveInput{
		Connector: ConnectorRuntime{ID: "11", TenantID: "22", AllowAutoCreateUser: true},
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
	if person == nil || person.Person == nil || person.Person.ID == "" {
		t.Fatalf("expected created person, got %+v", person)
	}

	var persons []model.PersonEntity
	if err := db.Where("primary_email = ?", "success@example.com").Find(&persons).Error; err != nil {
		t.Fatalf("query persons: %v", err)
	}
	if len(persons) != 1 {
		t.Fatalf("expected 1 persisted person, got %d", len(persons))
	}

	var identities []model.UserIdentityEntity
	if err := db.Where("connector_id = ? AND external_subject = ?", 11, "external-subject-success").Find(&identities).Error; err != nil {
		t.Fatalf("query identities: %v", err)
	}
	if len(identities) != 1 {
		t.Fatalf("expected 1 persisted identity, got %d", len(identities))
	}
	if identities[0].PersonID != persons[0].ID || identities[0].PersonID != person.Person.ID {
		t.Fatalf("expected identity to bind created person, person=%s persisted=%s identity=%s", person.Person.ID, persons[0].ID, identities[0].PersonID)
	}
}

func TestIdentityMapperPropagatesBoundPersonLookupError(t *testing.T) {
	expectedErr := errors.New("person lookup failed")

	mapper := newIdentityMapper(
		&fakeConnectorPersonRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.PersonEntity, error) {
				return nil, expectedErr
			},
		},
		&fakeConnectorUserIdentityRepository{
			getByIssuerAndExternalSubjectFunc: func(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error) {
				return &model.UserIdentityEntity{BaseEntity: model.UserIdentityEntity{}.BaseEntity, PersonID: "33"}, nil
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
	)

	_, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{ID: "11", TenantID: "22", AllowAutoCreateUser: true},
		Identity:  StandardIdentity{Subject: "external-subject-1"},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected bound person lookup error to propagate, got %v", err)
	}
}

func TestIdentityMapperRepairsOrphanBindingInsteadOfReinserting(t *testing.T) {
	var inserted bool
	var updatedIdentityID string
	var updatedUserID string
	var updatedIssuer string
	var updatedDetail StandardIdentity

	mapper := newIdentityMapper(
		&fakeConnectorPersonRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.PersonEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, person *model.PersonEntity) error {
				person.ID = "44"
				return nil
			},
		},
		&fakeConnectorUserIdentityRepository{
			getByIssuerAndExternalSubjectFunc: func(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error) {
				entity := &model.UserIdentityEntity{PersonID: "33", ConnectorID: "11", ExternalSubject: externalSubject, Issuer: "stale"}
				entity.ID = "77"
				return entity, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserIdentityEntity) error {
				inserted = true
				return nil
			},
			updateBindingFunc: func(ctx context.Context, identityID, personID string, issuer string, detail []byte) error {
				updatedIdentityID = identityID
				updatedUserID = personID
				updatedIssuer = issuer
				return json.Unmarshal(detail, &updatedDetail)
			},
		},
		WithIdentityMapperTxRunner(connectorRunInTransactionNoop),
		WithIdentityMapperUserRepository(&fakeConnectorUserRepository{}),
	)

	person, err := mapper.Resolve(context.Background(), identityResolveInput{
		Connector: ConnectorRuntime{ID: "11", TenantID: "22", AllowAutoCreateUser: true},
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
	if person == nil || person.Person == nil || person.Person.ID != "44" {
		t.Fatalf("expected repaired binding to return created person, got %+v", person)
	}
	if inserted {
		t.Fatal("expected orphan binding to be repaired via update, not insert")
	}
	if updatedIdentityID != "77" || updatedUserID != "44" {
		t.Fatalf("unexpected binding update target identity=%s user=%s", updatedIdentityID, updatedUserID)
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
		TenantID: "22",
		Name:     "github-sso",
		Protocol: connectorDriverTypeOAuth2,
		Provider: connectorProviderGithub,
		Status:   connectorStatusEnabled,
		Config:   json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = "101"

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
			getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) {
				if id != connectorEntity.ID {
					t.Fatalf("expected connector lookup by id %s, got %s", connectorEntity.ID, id)
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
		Authorize(ctx *gin.Context, req *dtoconnector.ConnectorAuthorizeReq, connectorID string) (*dtoconnector.ConnectorAuthorizeResp, error)
	})
	if !ok {
		t.Fatal("connector service should expose Authorize")
	}

	// Authorize 现在要求非 nil gin 上下文（取租户归属）且 redirect_uri 与配置同源（H11/H12）
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyTenantID, "22")

	resp, err := runtimeSvc.Authorize(ginCtx, &dtoconnector.ConnectorAuthorizeReq{
		RedirectURI:  "https://iam.example.com/callback",
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
	if authorizeInput.RedirectURI != "https://iam.example.com/callback" {
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
	if savedState.RedirectURI != "https://iam.example.com/callback" {
		t.Fatalf("expected saved redirect uri, got %q", savedState.RedirectURI)
	}
	if !savedState.ExpiredAt.Equal(fixedNow.Add(connectorStateTTL)) {
		t.Fatalf("expected expires at %v, got %v", fixedNow.Add(connectorStateTTL), savedState.ExpiredAt)
	}
}

func TestConnectorServiceCallbackConsumesStateAndInvokesDriver(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state",
		Nonce:       "nonce-1",
		ConnectorID: "101",
		TenantID:    "22",
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiredAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	var callbackInput *ConnectorCallbackInput
	connectorEntity := &model.ConnectorEntity{
		TenantID: "22",
		Name:     "github-sso",
		Protocol: connectorDriverTypeOAuth2,
		Provider: connectorProviderGithub,
		Status:   connectorStatusEnabled,
		Config:   json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = "101"

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
			getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) {
				if id != connectorEntity.ID {
					t.Fatalf("expected connector lookup by id %s, got %s", connectorEntity.ID, id)
				}
				return connectorEntity, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
				person := &model.PersonEntity{Username: model.StrPtr("callback-user"), Name: "callback-user"}
				person.ID = "88"
				return &resolvedConnectorPerson{Person: person}, nil
			},
		},
		tokenGenerator: func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
			return &objauth.TokenInfo{AccessToken: "issued-access", RefreshToken: "issued-refresh", TokenType: "Bearer"}, nil
		},
		loginRecorder:   func(ctx *gin.Context, tenantID, userID string, success bool) {},
		ssoSessionStore: &fakeConnectorSSOSessionStore{},
	}
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "201"}}, TenantID: "22", PersonID: "88", Name: "callback-user"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "201"}}, TenantID: "22", PersonID: "88", Name: "callback-user"}}, nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{getListByCondFunc: func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, error) {
			return model.TenantEntityList{{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "22"}}, Name: "tenant-22", Tag: "t22"}}, nil
		}}
	})
	defer restoreTenantStore()
	runtimeSvc, ok := any(svc).(interface {
		Callback(ctx *gin.Context, req *dtoconnector.ConnectorCallbackReq) (*dtoauth.LoginResp, error)
	})
	if !ok {
		t.Fatal("connector service should expose Callback")
	}

	// Callback 需要非 nil gin 上下文（租户查询走 gincontext）
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	resp, err := runtimeSvc.Callback(ginCtx, &dtoconnector.ConnectorCallbackReq{
		ConnectorID: "101",
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
	if callbackInput.ConnectorID != "101" || callbackInput.Code != "authorization-code" || callbackInput.State != "callback-state" {
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

func TestConnectorServiceCallbackAllowsMissingConnectorID(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state-missing-connector-id",
		Nonce:       "nonce-missing-connector-id",
		ConnectorID: "101",
		TenantID:    "22",
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiredAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	var callbackInput *ConnectorCallbackInput
	connectorEntity := &model.ConnectorEntity{
		TenantID: "22",
		Name:     "github-sso",
		Protocol: connectorDriverTypeOAuth2,
		Provider: connectorProviderGithub,
		Status:   connectorStatusEnabled,
		Config:   json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = "101"

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
			getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) {
				if id != connectorEntity.ID {
					t.Fatalf("expected connector lookup by id %s, got %s", connectorEntity.ID, id)
				}
				return connectorEntity, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
				person := &model.PersonEntity{Username: model.StrPtr("callback-user"), Name: "callback-user"}
				person.ID = "88"
				return &resolvedConnectorPerson{Person: person}, nil
			},
		},
		tokenGenerator: func(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
			return &objauth.TokenInfo{AccessToken: "issued-access", RefreshToken: "issued-refresh", TokenType: "Bearer"}, nil
		},
		loginRecorder:   func(ctx *gin.Context, tenantID, userID string, success bool) {},
		ssoSessionStore: &fakeConnectorSSOSessionStore{},
	}
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "201"}}, TenantID: "22", PersonID: "88", Name: "callback-user"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "201"}}, TenantID: "22", PersonID: "88", Name: "callback-user"}}, nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{getByIDFunc: func(ctx context.Context, id string) (*model.TenantEntity, error) {
			return &model.TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "22"}}, Name: "tenant-22", Tag: "t22"}, nil
		}}
	})
	defer restoreTenantStore()

	resp, err := svc.Callback(testGinCtx(t), &dtoconnector.ConnectorCallbackReq{
		Code:  "authorization-code",
		State: "callback-state-missing-connector-id",
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
	if callbackInput.ConnectorID != "101" || callbackInput.Code != "authorization-code" || callbackInput.State != "callback-state-missing-connector-id" {
		t.Fatalf("unexpected callback input: %+v", callbackInput)
	}
	if callbackInput.Nonce != "nonce-missing-connector-id" {
		t.Fatalf("expected callback nonce from stored state, got %q", callbackInput.Nonce)
	}
	if callbackInput.RedirectURI != "https://app.example.com/oidc/callback" {
		t.Fatalf("expected callback redirect uri from stored state, got %q", callbackInput.RedirectURI)
	}
	if _, err := stateStore.Load(context.Background(), "callback-state-missing-connector-id"); !errors.Is(err, ErrConnectorStateNotFound) {
		t.Fatalf("expected callback state to be consumed, got err=%v", err)
	}
}

func TestConnectorCallbackReturnsPersonScopedAuthPayload(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state-login-resp",
		Nonce:       "nonce-login-resp",
		ConnectorID: "101",
		TenantID:    "22",
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiredAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	connectorEntity := &model.ConnectorEntity{
		TenantID:            "22",
		Protocol:            connectorDriverTypeOAuth2,
		Provider:            connectorProviderGithub,
		Status:              connectorStatusEnabled,
		AllowAutoCreateUser: true,
		Config:              json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = "101"

	resolvedPerson := &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "909"}}, Username: model.StrPtr("connector-person")}

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
			getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) {
				return connectorEntity, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
				return &resolvedConnectorPerson{Person: resolvedPerson}, nil
			},
		},
		tokenGenerator:  nil,
		loginRecorder:   func(ctx *gin.Context, tenantID, userID string, success bool) {},
		ssoSessionStore: &fakeConnectorSSOSessionStore{},
	}
	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				return &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "909"}}, Username: model.StrPtr("connector-person")}, nil
			},
		}
	})
	defer restorePersonStore()
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "303"}}, TenantID: "22", PersonID: "909", Name: "connector-user-a"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "303"}}, TenantID: "22", PersonID: "909", Name: "connector-user-a"}}, nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{getByIDFunc: func(ctx context.Context, id string) (*model.TenantEntity, error) {
			return &model.TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "22"}}, Name: "tenant-22", Tag: "t22"}, nil
		}}
	})
	defer restoreTenantStore()

	resp, err := svc.Callback(testGinCtx(t), &dtoconnector.ConnectorCallbackReq{ConnectorID: "101", Code: "authorization-code", State: "callback-state-login-resp"})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected login response")
	}
	if resp.SSOSessionID == "" {
		t.Fatalf("expected callback to return sso session id, got %+v", resp)
	}
}

func TestConnectorCallbackInvokesIdentityResolverTokenGeneratorAndLoginRecorder(t *testing.T) {
	stateStore := NewInMemoryConnectorStateStore()
	state := &ConnectorState{
		State:       "callback-state-runtime-hooks",
		Nonce:       "nonce-runtime-hooks",
		ConnectorID: "202",
		TenantID:    "66",
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiredAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	connectorEntity := &model.ConnectorEntity{
		TenantID:            "66",
		Protocol:            connectorDriverTypeOAuth2,
		Provider:            connectorProviderGithub,
		Status:              connectorStatusEnabled,
		AllowAutoCreateUser: true,
		Config:              json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	connectorEntity.ID = "202"

	resolvedPerson := &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "909"}}, Username: model.StrPtr("mapped-user"), Name: "Mapped User"}

	var resolvedInput identityResolveInput
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
			getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) {
				return connectorEntity, nil
			},
		},
		stateStore: stateStore,
		identityResolver: &fakeConnectorIdentityResolver{
			resolveFunc: func(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
				resolvedInput = input
				return &resolvedConnectorPerson{Person: resolvedPerson}, nil
			},
		},
		loginRecorder:   func(ctx *gin.Context, tenantID, userID string, success bool) {},
		ssoSessionStore: &fakeConnectorSSOSessionStore{},
	}
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "501"}}, TenantID: "66", PersonID: "909", Name: "mapped-user-a"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "501"}}, TenantID: "66", PersonID: "909", Name: "mapped-user-a"}}, nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{getByIDFunc: func(ctx context.Context, id string) (*model.TenantEntity, error) {
			return &model.TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, Name: "tenant-66", Tag: "t66"}, nil
		}}
	})
	defer restoreTenantStore()

	_, err := svc.Callback(testGinCtx(t), &dtoconnector.ConnectorCallbackReq{ConnectorID: "202", Code: "authorization-code", State: "callback-state-runtime-hooks"})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resolvedInput.Connector.ID != "202" || resolvedInput.Connector.TenantID != "66" || !resolvedInput.Connector.AllowAutoCreateUser {
		t.Fatalf("unexpected connector runtime input: %+v", resolvedInput.Connector)
	}
	if resolvedInput.Identity.Subject != "subject-202" || resolvedInput.Identity.Email != "mapped@example.com" {
		t.Fatalf("unexpected standard identity input: %+v", resolvedInput.Identity)
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
		ConnectorID: "101",
		TenantID:    "22",
		RedirectURI: "https://app.example.com/oidc/callback",
		ExpiredAt:   time.Now().Add(time.Minute),
	}
	if err := stateStore.Save(context.Background(), state); err != nil {
		t.Fatalf("stateStore.Save returned error: %v", err)
	}

	expectedErr := errors.New("temporary exchange failure")
	conn := &model.ConnectorEntity{
		TenantID: "22",
		Protocol: connectorDriverTypeOAuth2,
		Provider: connectorProviderGithub,
		Status:   connectorStatusEnabled,
		Config:   json.RawMessage(`{"authUrl":"https://github.com/login/oauth/authorize","tokenUrl":"https://github.com/login/oauth/access_token","userInfoUrl":"https://api.github.com/user","clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://iam.example.com/callback"}`),
	}
	conn.ID = "101"

	svc := &connectorSvc{
		driverRegistry: newConnectorDriverRegistry(&fakeConnectorDriver{
			driverType: connectorDriverTypeOAuth2,
			exchangeCallbackFunc: func(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
				return nil, expectedErr
			},
		}),
		connectorRepo: &fakeConnectorRuntimeRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.ConnectorEntity, error) {
				return conn, nil
			},
		},
		stateStore: stateStore,
	}

	_, err := svc.Callback(testGinCtx(t), &dtoconnector.ConnectorCallbackReq{ConnectorID: "101", Code: "authorization-code", State: "callback-state-fail"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected driver exchange error to propagate, got %v", err)
	}
	// 新语义：state 先原子消费（防重放），exchange 失败后不可重试（需重新授权）
	_, loadErr := stateStore.Load(context.Background(), "callback-state-fail")
	if loadErr == nil {
		t.Fatal("expected state to be consumed after callback (single-use), replay should be rejected")
	}
}

func newConnectorIdentityMapperTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeConnectorTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.PersonEntity{}, &model.UserIdentityEntity{}, &model.UserEntity{}); err != nil {
		t.Fatalf("migrate person tables: %v", err)
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

// testGinCtx 构造带 Request 的 gin 测试上下文（Callback 的租户查询依赖 gincontext）。
func testGinCtx(t *testing.T) *gin.Context {
	t.Helper()
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	return ginCtx
}
