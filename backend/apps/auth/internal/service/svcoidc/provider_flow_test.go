package svcoidc

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"gorm.io/gorm"
)

func TestFullOIDCCodeFlow(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	issuer := "http://localhost:8099/v1/iam/oidc"
	appconfig.Conf = &appconfig.Config{
		JWT: appconfig.JWT{SignKey: "test-sign-key"},
		OIDC: appconfig.OIDC{
			Issuer:        issuer,
			AllowInsecure: true,
		},
	}

	provider, err := SetupOIDCProvider(issuer)
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}

	ctx := t.Context()

	authReq, err := provider.Storage.CreateAuthRequest(ctx, &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-test",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
		Nonce:        "nonce-test",
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, error) {
			return &model.PersonEntity{Model: gorm.Model{ID: 88}}, &model.UserEntity{Model: gorm.Model{ID: 66}, TenantID: 1, PersonID: 88}, nil
		}},
	}

	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/v1/iam/oidc/login", nil)
	res, err := svc.CompleteLogin(ginCtx, &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "person@example.com",
		Password:      "Password1",
	})
	if err != nil {
		t.Fatalf("CompleteLogin failed: %v", err)
	}

	expectedSuffix := "/authorize/callback?id=" + authReq.GetID()
	if !strings.HasSuffix(res.ContinueURL, expectedSuffix) {
		t.Fatalf("continueURL %q does not end with %q", res.ContinueURL, expectedSuffix)
	}

	code, err := op.CreateAuthRequestCode(ctx, authReq, provider.Storage, provider.Provider.Crypto())
	if err != nil {
		t.Fatalf("CreateAuthRequestCode failed: %v", err)
	}
	if code == "" {
		t.Fatal("expected non-empty authorization code")
	}

	foundReq, err := provider.Storage.AuthRequestByCode(ctx, code)
	if err != nil {
		t.Fatalf("AuthRequestByCode failed: %v", err)
	}
	if foundReq.GetID() != authReq.GetID() {
		t.Fatalf("expected auth request ID %s, got %s", authReq.GetID(), foundReq.GetID())
	}

	if !foundReq.Done() {
		t.Fatal("expected auth request to remain completed after code generation")
	}
	if foundReq.GetSubject() != "person:88" {
		t.Fatalf("expected subject person:88, got %q", foundReq.GetSubject())
	}
}
