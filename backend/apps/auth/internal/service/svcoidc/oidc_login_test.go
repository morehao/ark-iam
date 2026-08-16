package svcoidc

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/assert"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

type fakePasswordAuthenticator struct {
	authenticate     func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error)
	tenantsForPerson func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error)
}

func (f *fakePasswordAuthenticator) AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
	return f.authenticate(ctx, identifier, password)
}

func (f *fakePasswordAuthenticator) TenantsForPerson(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
	if f.tenantsForPerson != nil {
		return f.tenantsForPerson(ctx, personID)
	}
	return nil, nil
}

type fakeSSOSessionStore struct {
	validatedPersonID string
	sessionAMR        []string
}

var _ sso.SSOSessionStore = (*fakeSSOSessionStore)(nil)

func (f *fakeSSOSessionStore) CreateSession(ctx context.Context, personID string, amr []string) (string, error) {
	return "session-" + fmt.Sprint(personID), nil
}
func (f *fakeSSOSessionStore) ValidateSession(ctx context.Context, sessionID string) (string, error) {
	return f.validatedPersonID, nil
}
func (f *fakeSSOSessionStore) SessionAMR(ctx context.Context, sessionID string) []string {
	return f.sessionAMR
}
func (f *fakeSSOSessionStore) RevokeSession(ctx context.Context, sessionID string) error { return nil }
func (f *fakeSSOSessionStore) RevokeSessionsByPersonID(ctx context.Context, personID string) error {
	return nil
}
func (f *fakeSSOSessionStore) HasActiveSession(ctx context.Context, personID string) (bool, error) {
	return false, nil
}

func TestCompleteLoginBySessionHonorsAuthRequestTenantHint(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "7", false); err != nil {
		t.Fatalf("storing tenant hint via CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
			if personID != "88" {
				t.Fatalf("expected personID 88, got %s", personID)
			}
			return []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}, {TenantID: "7", Name: "tenant-7"}}, nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{validatedPersonID: "88"},
	}

	res, err := svc.CompleteLoginBySession(testsetup.NewCtx(), request.GetID(), "session-x")
	if err != nil {
		t.Fatalf("CompleteLoginBySession failed: %v", err)
	}
	if res != "http://localhost:8099/oidc/authorize/callback?id="+request.GetID() {
		t.Fatalf("unexpected continueURL: %q", res)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	completedReq, ok := updated.(*oidcop.AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if !completedReq.Done() {
		t.Fatal("expected auth request to be completed")
	}
	if completedReq.TenantID != "7" {
		t.Fatalf("expected SSO to honor tenant hint 7, got %s", completedReq.TenantID)
	}
}

func TestCompleteLoginBySessionFallsBackWhenHintNotInPersonsTenants(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "99", false); err != nil {
		t.Fatalf("storing forged tenant hint via CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
			return []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}}, nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{validatedPersonID: "88"},
	}

	if _, err := svc.CompleteLoginBySession(testsetup.NewCtx(), request.GetID(), "session-x"); err != nil {
		t.Fatalf("CompleteLoginBySession failed: %v", err)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	completedReq, ok := updated.(*oidcop.AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if !completedReq.Done() {
		t.Fatal("expected auth request to be completed")
	}
	if completedReq.TenantID != "3" {
		t.Fatalf("expected forged hint 99 to fall back to tenants[0]=3, got %s", completedReq.TenantID)
	}
}

func TestCompleteLoginBySessionRejectsHintOnTenantLookupError(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "99", false); err != nil {
		t.Fatalf("storing forged tenant hint via CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
			return nil, assert.AnError
		}},
		ssoSessionStore: &fakeSSOSessionStore{validatedPersonID: "88"},
	}

	if _, err := svc.CompleteLoginBySession(testsetup.NewCtx(), request.GetID(), "session-x"); err != nil {
		t.Fatalf("CompleteLoginBySession failed: %v", err)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	completedReq, ok := updated.(*oidcop.AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if !completedReq.Done() {
		t.Fatal("expected auth request to be completed")
	}
	if completedReq.TenantID != "" {
		t.Fatalf("expected forged hint to be dropped (tenantID 0) when tenant lookup errors, got %s", completedReq.TenantID)
	}
}

func TestCompleteLoginReturnsContinueURLAndCompletesRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			if identifier != "person@example.com" || password != "Password1" {
				t.Fatalf("unexpected credentials identifier=%q password=%q", identifier, password)
			}
			return &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "88"}}}, &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, TenantID: "1", PersonID: "88"}, []objauth.TenantOption{{TenantID: "1", Name: "tenant-1"}}, nil
		}},
	}

	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/login", nil)
	res, err := svc.CompleteLogin(ginCtx, &dtooidc.OIDCLoginReq{
		AuthRequestID: request.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",
	})
	if err != nil {
		t.Fatalf("CompleteLogin failed: %v", err)
	}
	if res.ContinueURL != "http://localhost:8099/oidc/authorize/callback?id="+request.GetID() {
		t.Fatalf("unexpected continueURL: %q", res.ContinueURL)
	}
	if res.TenantID != "1" {
		t.Fatalf("expected tenantID 1, got %s", res.TenantID)
	}
	if len(res.Tenants) != 1 || res.Tenants[0].TenantID != "1" {
		t.Fatalf("unexpected tenants: %#v", res.Tenants)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	if !updated.Done() {
		t.Fatal("expected auth request to be completed")
	}
	if updated.GetSubject() != "person:88" {
		t.Fatalf("expected completed subject person:88, got %q", updated.GetSubject())
	}
	completedReq, ok := updated.(*oidcop.AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if completedReq.TenantID != "1" {
		t.Fatalf("expected completed auth request tenantID 1, got %s", completedReq.TenantID)
	}
}

func TestCompleteLoginMultiTenantRequiresSelection(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			return &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "88"}}}, &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, TenantID: "3", PersonID: "88"}, []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}, {TenantID: "7", Name: "tenant-7"}}, nil
		}},
	}

	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/login", nil)
	res, err := svc.CompleteLogin(ginCtx, &dtooidc.OIDCLoginReq{
		AuthRequestID: request.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",
	})
	if err != nil {
		t.Fatalf("CompleteLogin failed: %v", err)
	}
	if !res.RequiresTenantSelection {
		t.Fatal("expected requiresTenantSelection=true for multi-tenant login")
	}
	if len(res.Tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %#v", res.Tenants)
	}
	if res.ContinueURL != "" {
		t.Fatalf("expected empty continueURL for multi-tenant login, got %q", res.ContinueURL)
	}
	if res.SessionID != "" {
		t.Fatalf("expected no sessionID for multi-tenant login, got %q", res.SessionID)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	if updated.Done() {
		t.Fatal("expected multi-tenant auth request NOT to be completed (no code until tenant chosen)")
	}
	if updated.GetSubject() != "person:88" {
		t.Fatalf("expected stored subject person:88, got %q", updated.GetSubject())
	}
}

func TestCompleteLoginHonorsTenantHint(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "7", false); err != nil {
		t.Fatalf("storing tenant hint via CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			return &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "88"}}}, &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, TenantID: "3", PersonID: "88"}, []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}, {TenantID: "7", Name: "tenant-7"}}, nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{},
	}

	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/login", nil)
	res, err := svc.CompleteLogin(ginCtx, &dtooidc.OIDCLoginReq{
		AuthRequestID: request.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",
	})
	if err != nil {
		t.Fatalf("CompleteLogin failed: %v", err)
	}
	if res.RequiresTenantSelection {
		t.Fatal("expected no tenant selection when hint is a valid membership")
	}
	if res.TenantID != "7" {
		t.Fatalf("expected tenant hint 7 to override multi-tenant auto-selection, got %s", res.TenantID)
	}
	if res.ContinueURL != "http://localhost:8099/oidc/authorize/callback?id="+request.GetID() {
		t.Fatalf("unexpected continueURL: %q", res.ContinueURL)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	completedReq, ok := updated.(*oidcop.AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if !completedReq.Done() {
		t.Fatal("expected auth request to be completed with valid tenant hint")
	}
	if completedReq.TenantID != "7" {
		t.Fatalf("expected completed auth request tenantID 7, got %s", completedReq.TenantID)
	}
}

func TestCompleteLoginIgnoresForgedTenantHint(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "99", false); err != nil {
		t.Fatalf("storing forged tenant hint via CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			return &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "88"}}}, &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, TenantID: "3", PersonID: "88"}, []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}}, nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{},
	}

	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/login", nil)
	res, err := svc.CompleteLogin(ginCtx, &dtooidc.OIDCLoginReq{
		AuthRequestID: request.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",
	})
	if err != nil {
		t.Fatalf("CompleteLogin failed: %v", err)
	}
	if res.TenantID != "3" {
		t.Fatalf("expected forged hint 99 to fall back to auto-picked tenant 3, got %s", res.TenantID)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	completedReq, ok := updated.(*oidcop.AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if !completedReq.Done() {
		t.Fatal("expected auth request to be completed")
	}
	if completedReq.TenantID != "3" {
		t.Fatalf("expected completed auth request tenantID 3, got %s", completedReq.TenantID)
	}
}

func TestSelectTenantWritesTenantAndReturnsContinueURL(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "", false); err != nil {
		t.Fatalf("CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
			if personID != "88" {
				t.Fatalf("expected personID 88, got %s", personID)
			}
			return []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}, {TenantID: "7", Name: "tenant-7"}}, nil
		}},
	}

	res, err := svc.SelectTenant(testsetup.NewCtx(), request.GetID(), "7")
	if err != nil {
		t.Fatalf("SelectTenant failed: %v", err)
	}
	if res.ContinueURL != "http://localhost:8099/oidc/authorize/callback?id="+request.GetID() {
		t.Fatalf("unexpected continueURL: %q", res.ContinueURL)
	}
	if res.TenantID != "7" {
		t.Fatalf("expected tenantID 7, got %s", res.TenantID)
	}
	if len(res.Tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %#v", res.Tenants)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	if !updated.Done() {
		t.Fatal("expected auth request to be completed after tenant selection")
	}
	completedReq, ok := updated.(*oidcop.AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if completedReq.TenantID != "7" {
		t.Fatalf("expected completed auth request tenantID 7, got %s", completedReq.TenantID)
	}
}

func TestSelectTenantRejectsTenantNotBelongingToPerson(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "", false); err != nil {
		t.Fatalf("CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
			return []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}}, nil
		}},
	}

	if _, err := svc.SelectTenant(testsetup.NewCtx(), request.GetID(), "7"); err == nil {
		t.Fatal("expected error when selecting a tenant not belonging to the person")
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	if updated.Done() {
		t.Fatal("expected auth request NOT completed when tenant selection is rejected")
	}
}

func TestSelectTenantRejectsAlreadyDoneRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	request, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	callCount := 0
	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
			callCount++
			return []objauth.TenantOption{{TenantID: "3", Name: "tenant-3"}, {TenantID: "7", Name: "tenant-7"}}, nil
		}},
		ssoSessionStore: &fakeSSOSessionStore{},
	}

	authTime := time.Now()
	if err := provider.Storage.CompleteAuthRequest(t.Context(), request.GetID(), "person:88", authTime, []string{"pwd"}, "", "3", true); err != nil {
		t.Fatalf("CompleteAuthRequest(done=true) failed: %v", err)
	}

	if _, err := svc.SelectTenant(testsetup.NewCtx(), request.GetID(), "3"); err == nil {
		t.Fatalf("expected SelectTenant to reject an already-done request")
	}
	if callCount != 0 {
		t.Fatalf("expected tenants lookup NOT to run on done request, ran %d times", callCount)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), request.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	if !updated.Done() {
		t.Fatal("expected auth request to remain done")
	}
}
