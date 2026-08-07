package svcoidc

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"gorm.io/gorm"
)

type fakePasswordAuthenticator struct {
	authenticate     func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error)
	tenantsForPerson func(ctx *gin.Context, personID uint) ([]objauth.TenantOption, error)
}

func (f *fakePasswordAuthenticator) AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
	return f.authenticate(ctx, identifier, password)
}

func (f *fakePasswordAuthenticator) TenantsForPerson(ctx *gin.Context, personID uint) ([]objauth.TenantOption, error) {
	if f.tenantsForPerson != nil {
		return f.tenantsForPerson(ctx, personID)
	}
	return nil, nil
}

func TestCompleteLoginReturnsContinueURLAndCompletesRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider(appconfig.Conf.OIDC.Issuer)
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
			return &model.PersonEntity{Model: gorm.Model{ID: 88}}, &model.UserEntity{Model: gorm.Model{ID: 66}, TenantID: 1, PersonID: 88}, []objauth.TenantOption{{TenantID: 1, Name: "tenant-1"}}, nil
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
	if res.TenantID != 1 {
		t.Fatalf("expected tenantID 1, got %d", res.TenantID)
	}
	if len(res.Tenants) != 1 || res.Tenants[0].TenantID != 1 {
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
	completedReq, ok := updated.(*AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if completedReq.TenantID != 1 {
		t.Fatalf("expected completed auth request tenantID 1, got %d", completedReq.TenantID)
	}
}

func TestCompleteLoginMultiTenantRequiresSelection(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider(appconfig.Conf.OIDC.Issuer)
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
			return &model.PersonEntity{Model: gorm.Model{ID: 88}}, &model.UserEntity{Model: gorm.Model{ID: 66}, TenantID: 3, PersonID: 88}, []objauth.TenantOption{{TenantID: 3, Name: "tenant-3"}, {TenantID: 7, Name: "tenant-7"}}, nil
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

func TestSelectTenantWritesTenantAndReturnsContinueURL(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider(appconfig.Conf.OIDC.Issuer)
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
	if err := provider.Storage.CompleteAuthRequest(request.GetID(), "person:88", authTime, []string{"pwd"}, "", 0, false); err != nil {
		t.Fatalf("CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID uint) ([]objauth.TenantOption, error) {
			if personID != 88 {
				t.Fatalf("expected personID 88, got %d", personID)
			}
			return []objauth.TenantOption{{TenantID: 3, Name: "tenant-3"}, {TenantID: 7, Name: "tenant-7"}}, nil
		}},
	}

	res, err := svc.SelectTenant(t.Context(), request.GetID(), 7)
	if err != nil {
		t.Fatalf("SelectTenant failed: %v", err)
	}
	if res.ContinueURL != "http://localhost:8099/oidc/authorize/callback?id="+request.GetID() {
		t.Fatalf("unexpected continueURL: %q", res.ContinueURL)
	}
	if res.TenantID != 7 {
		t.Fatalf("expected tenantID 7, got %d", res.TenantID)
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
	completedReq, ok := updated.(*AuthRequest)
	if !ok {
		t.Fatalf("expected *AuthRequest, got %T", updated)
	}
	if completedReq.TenantID != 7 {
		t.Fatalf("expected completed auth request tenantID 7, got %d", completedReq.TenantID)
	}
}

func TestSelectTenantRejectsTenantNotBelongingToPerson(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:3000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider(appconfig.Conf.OIDC.Issuer)
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
	if err := provider.Storage.CompleteAuthRequest(request.GetID(), "person:88", authTime, []string{"pwd"}, "", 0, false); err != nil {
		t.Fatalf("CompleteAuthRequest(done=false) failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{tenantsForPerson: func(ctx *gin.Context, personID uint) ([]objauth.TenantOption, error) {
			return []objauth.TenantOption{{TenantID: 3, Name: "tenant-3"}}, nil
		}},
	}

	if _, err := svc.SelectTenant(t.Context(), request.GetID(), 7); err == nil {
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
