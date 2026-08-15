package svcoidc

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func TestCreateAuthRequestStoresTenant(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "", "5")
	require.NoError(t, err)
	reqTyped, ok := req.(*AuthRequest)
	require.True(t, ok, "expected *AuthRequest from store")
	assert.Equal(t, "5", reqTyped.GetTenantID())

	found, err := store.AuthRequestByID(context.Background(), req.GetID())
	require.NoError(t, err)
	assert.Equal(t, "5", found.(*AuthRequest).GetTenantID())
}

func TestCreateAuthRequestReadsTenantHintFromContext(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.WithValue(context.Background(), TenantHintKey, "5")
	req, err := NewRedisProtocolStateStore().CreateAuthRequest(ctx, &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "", "5")
	require.NoError(t, err)
	assert.Equal(t, "5", req.(*AuthRequest).GetTenantID())
}

func TestMiddlewareToStoragePropagatesTenant(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	store := NewRedisProtocolStateStore()
	storage := NewOIDCStorage(store, nil, nil, "")

	withValue := context.WithValue(context.Background(), TenantHintKey, "7")
	res, err := storage.CreateAuthRequest(withValue, &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)
	assert.Equal(t, "7", res.(*AuthRequest).GetTenantID())
}
