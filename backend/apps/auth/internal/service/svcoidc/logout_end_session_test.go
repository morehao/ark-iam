package svcoidc

import (
	"context"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"gorm.io/gorm"
)

func TestTerminateSessionFromRequestEnqueuesPreciseSidJobs(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 80}, TenantID: 1, PersonID: 88},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	// 登记两个会话：sid-a 与 sid-b，person 88
	sloStore := sso.NewSLOStore()
	regA := sso.LogoutRegistration{OIDCSessionID: "at-a", ClientID: "client-a", UserID: "person:88", BackChannelLogoutURI: "https://a/bc"}
	regB := sso.LogoutRegistration{OIDCSessionID: "at-b", ClientID: "client-b", UserID: "person:88", BackChannelLogoutURI: "https://b/bc"}
	if err := sloStore.Register(ctx, "sid-a", regA); err != nil {
		t.Fatalf("register sid-a: %v", err)
	}
	if err := sloStore.Register(ctx, "sid-b", regB); err != nil {
		t.Fatalf("register sid-b: %v", err)
	}

	// 带 sid 的登出：精准注销 sid-a，只应入队 client-a
	endSession := &op.EndSessionRequest{
		UserID: "person:88",
		IDTokenHintClaims: &oidc.IDTokenClaims{
			SessionID: "sid-a",
		},
		RedirectURI: "https://rp.example.com/logged-out",
	}
	if _, err := storage.TerminateSessionFromRequest(ctx, endSession); err != nil {
		t.Fatalf("TerminateSessionFromRequest failed: %v", err)
	}

	// 队列应有一条 sid-a 的任务
	job, ok, err := sso.DequeueLogout(ctx, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if !ok {
		t.Fatal("expected a logout job")
	}
	if job.SessionID != "sid-a" {
		t.Fatalf("expected job SessionID sid-a, got %q", job.SessionID)
	}
	if job.ClientID != "client-a" {
		t.Fatalf("expected job client client-a, got %q", job.ClientID)
	}

	// 不应再有多余任务
	if _, ok, _ = sso.DequeueLogout(ctx, 200*time.Millisecond); ok {
		t.Fatal("expected no additional logout job")
	}
}
