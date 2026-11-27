package oidcop

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// TestAuthCodeIssuerNamespaceIsolation 覆盖 F4/AC-5：A issuer 的 code 在 B issuer 不可消费。
func TestAuthCodeIssuerNamespaceIsolation(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	store := NewRedisProtocolStateStore()
	issuerA := "https://login.acme.com/oidc"
	issuerB := "https://login.initech.com/oidc"

	ctxA := op.ContextWithIssuer(context.Background(), issuerA)
	// 用户已完成认证（userID 非空 → DoneFlag=true），可发 code
	ar, err := store.CreateAuthRequest(ctxA, &oidc.AuthRequest{
		ClientID: "tenant_console", RedirectURI: "https://login.acme.com/admin/callback",
		Scopes: []string{oidc.ScopeOpenID}, ResponseType: oidc.ResponseTypeCode, ResponseMode: oidc.ResponseModeQuery,
	}, "person:88", "tA")
	require.NoError(t, err)
	code := "iso-code-" + ar.GetID()
	// 在 issuerA 上下文保存 code
	require.NoError(t, store.SaveAuthCode(ctxA, ar.GetID(), code))

	// B issuer 上下文消费同一 code：应取不到（命名空间隔离）→ ErrCodeInvalid
	ctxB := op.ContextWithIssuer(context.Background(), issuerB)
	_, errB := store.AuthRequestByCode(ctxB, code)
	require.Error(t, errB, "A issuer 的 code 不应在 B issuer 可消费")

	// A issuer 上下文仍可正常消费
	got, errA := store.AuthRequestByCode(ctxA, code)
	require.NoError(t, errA)
	require.Equal(t, ar.GetID(), got.GetID())
}
