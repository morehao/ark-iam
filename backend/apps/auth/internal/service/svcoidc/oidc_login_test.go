package svcoidc

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"gorm.io/gorm"
)

type fakePasswordAuthenticator struct {
	authenticate func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, error)
}

func (f *fakePasswordAuthenticator) AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, error) {
	return f.authenticate(ctx, identifier, password)
}

func TestCompleteLoginReturnsContinueURLAndCompletesRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:           "http://localhost:8099/v1/iam/oidc",
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
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, error) {
			if identifier != "person@example.com" || password != "Password1" {
				t.Fatalf("unexpected credentials identifier=%q password=%q", identifier, password)
			}
			return &model.PersonEntity{Model: gorm.Model{ID: 88}}, &model.UserEntity{Model: gorm.Model{ID: 66}, TenantID: 1, PersonID: 88}, nil
		}},
	}

	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/v1/iam/oidc/login", nil)
	res, err := svc.CompleteLogin(ginCtx, &dtooidc.OIDCLoginReq{
		AuthRequestID: request.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",
	})
	if err != nil {
		t.Fatalf("CompleteLogin failed: %v", err)
	}
	if res.ContinueURL != "http://localhost:8099/v1/iam/oidc/authorize/callback?id="+request.GetID() {
		t.Fatalf("unexpected continueURL: %q", res.ContinueURL)
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
}
