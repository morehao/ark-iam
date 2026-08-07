package svcoidc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func TestCreateAndGetAuthRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
		Nonce:        "nonce-1",
	}, "")
	require.NoError(t, err)
	require.NotEmpty(t, req.GetID())
	assert.False(t, req.Done())

	found, err := store.AuthRequestByID(context.Background(), req.GetID())
	require.NoError(t, err)
	assert.Equal(t, req.GetID(), found.GetID())
	assert.Equal(t, "client-1", found.GetClientID())
}

func TestAuthRequestByIDNotFound(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	_, err := store.AuthRequestByID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestCompleteAuthRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	authTime := time.Unix(1710000000, 0)
	err = store.CompleteAuthRequest(req.GetID(), "person:88", authTime, []string{"pwd"}, "", 0, true)
	require.NoError(t, err)

	found, err := store.AuthRequestByID(context.Background(), req.GetID())
	require.NoError(t, err)
	assert.True(t, found.Done())
	assert.Equal(t, "person:88", found.GetSubject())
	assert.Equal(t, authTime.Unix(), found.GetAuthTime().Unix())
	assert.Equal(t, []string{"pwd"}, found.GetAMR())
}

func TestSaveAndConsumeCode(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	_ = store.CompleteAuthRequest(req.GetID(), "person:88", time.Now(), []string{"pwd"}, "", 0, true)

	code := fmt.Sprintf("auth-code-%d", time.Now().UnixNano())
	err = store.SaveAuthCode(context.Background(), req.GetID(), code)
	require.NoError(t, err)

	// Lookup by code (non-consuming)
	found, err := store.AuthRequestByCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, req.GetID(), found.GetID())
	assert.True(t, found.Done())

	// Consume the code
	found, err = store.ConsumeAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, req.GetID(), found.GetID())

	// After consume, AuthRequestByCode should return ErrCodeAlreadyUsed
	_, err = store.AuthRequestByCode(context.Background(), code)
	assert.ErrorIs(t, err, ErrCodeAlreadyUsed)
}

func TestDeleteAuthRequest(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	err = store.DeleteAuthRequest(context.Background(), req.GetID())
	require.NoError(t, err)

	_, err = store.AuthRequestByID(context.Background(), req.GetID())
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestConsumeCodeNotCompleted(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	req, err := store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{
		ClientID:     "client-1",
		RedirectURI:  "https://client.example.com/callback",
		State:        "state-1",
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	require.NoError(t, err)

	// Save code WITHOUT completing the auth request
	code := fmt.Sprintf("code-not-completed-%d", time.Now().UnixNano())
	err = store.SaveAuthCode(context.Background(), req.GetID(), code)
	require.NoError(t, err)

	_, err = store.AuthRequestByCode(context.Background(), code)
	assert.ErrorIs(t, err, ErrSessionNotCompleted)
}

func TestHealthCheck(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	store := NewRedisProtocolStateStore()
	err := store.Health(context.Background())
	require.NoError(t, err)
}
