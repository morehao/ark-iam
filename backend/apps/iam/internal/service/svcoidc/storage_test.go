package svcoidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	appconfig "github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func TestBuildOIDCSubject(t *testing.T) {
	if got := buildOIDCSubject(12); got != "person:12" {
		t.Fatalf("expected subject person:12, got %q", got)
	}
}

func TestParseOIDCSubject(t *testing.T) {
	personID, err := parseOIDCSubject("person:34")
	if err != nil {
		t.Fatalf("expected subject to parse, got err: %v", err)
	}
	if personID != 34 {
		t.Fatalf("expected personID 34, got %d", personID)
	}
}

func TestParseOIDCSubjectRejectsInvalidFormat(t *testing.T) {
	_, err := parseOIDCSubject("34")
	if err == nil {
		t.Fatal("expected invalid subject format to fail")
	}
	if !strings.Contains(err.Error(), "invalid oidc subject") {
		t.Fatalf("expected invalid subject error, got %v", err)
	}
}

func TestCompleteAuthRequestMarksRequestDone(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	storage := NewOIDCStorage(NewRedisProtocolStateStore(), NewPersistentStore(), privateKey, "test-key")
	req, err := storage.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
		Nonce:        "nonce-1",
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest failed: %v", err)
	}
	if req.Done() {
		t.Fatal("expected auth request to start incomplete")
	}

	authTime := time.Unix(1710000000, 0)
	err = storage.CompleteAuthRequest(req.GetID(), buildOIDCSubject(88), authTime, []string{"pwd"}, "", 0, true)
	if err != nil {
		t.Fatalf("CompleteAuthRequest failed: %v", err)
	}

	updated, err := storage.AuthRequestByID(context.Background(), req.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	if !updated.Done() {
		t.Fatal("expected auth request to be marked done")
	}
	if updated.GetSubject() != "person:88" {
		t.Fatalf("expected subject person:88, got %q", updated.GetSubject())
	}
	if !updated.GetAuthTime().Equal(authTime) {
		t.Fatalf("expected auth time %v, got %v", authTime, updated.GetAuthTime())
	}
	amr := updated.GetAMR()
	if len(amr) != 1 || amr[0] != "pwd" {
		t.Fatalf("expected amr [pwd], got %#v", amr)
	}
}

func TestOIDCClientLoginURLUsesConfiguredFrontend(t *testing.T) {
	appconfig.Conf = &appconfig.Config{OIDC: appconfig.OIDC{FrontendLoginURL: "https://console.example.com/oidc/login"}}
	client := NewOIDCClient(&model.OAuthClientEntity{ClientID: "client-1"})

	got := client.LoginURL("ar-1")
	if got != "https://console.example.com/oidc/login?authRequestID=ar-1" {
		t.Fatalf("expected configured login url, got %q", got)
	}
}

func TestSigningKeyUsesAsymmetricPrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	storage := NewOIDCStorage(nil, nil, privateKey, "test-key")
	key, err := storage.SigningKey(context.Background())
	if err != nil {
		t.Fatalf("SigningKey failed: %v", err)
	}
	if _, ok := key.Key().(*rsa.PrivateKey); !ok {
		t.Fatalf("expected rsa private key, got %T", key.Key())
	}
}

func TestKeySetExposesPublicKeyOnly(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	storage := NewOIDCStorage(nil, nil, privateKey, "test-key")
	keys, err := storage.KeySet(context.Background())
	if err != nil {
		t.Fatalf("KeySet failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected one key, got %d", len(keys))
	}
	if _, ok := keys[0].Key().(*rsa.PublicKey); !ok {
		t.Fatalf("expected rsa public key, got %T", keys[0].Key())
	}
}
