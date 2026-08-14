package sso

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

func TestLogoutRegistrationRegisterAndListBySessionID(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.Background()
	store := NewSLOStore()
	sid := "test-slo-session-1"

	reg := LogoutRegistration{
		OIDCSessionID:        "at-1",
		ClientID:             "client-a",
		UserID:               "person:42",
		BackChannelLogoutURI: "https://app-a.example.com/bc-logout",
	}
	require.NoError(t, store.Register(ctx, sid, reg))

	regs, err := store.ListBySessionID(ctx, sid)
	require.NoError(t, err)
	require.Len(t, regs, 1)
	require.Equal(t, "client-a", regs[0].ClientID)
	require.Equal(t, "https://app-a.example.com/bc-logout", regs[0].BackChannelLogoutURI)

	_ = store.Delete(ctx, sid, "at-1")

	regs, err = store.ListBySessionID(ctx, sid)
	require.NoError(t, err)
	require.Len(t, regs, 0)
}

func TestLogoutRegistrationDeleteOnlyMatchesOIDCSession(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.Background()
	store := NewSLOStore()
	sid := "test-slo-session-2"

	regA := LogoutRegistration{OIDCSessionID: "at-a", ClientID: "client-a", UserID: "person:7", BackChannelLogoutURI: "https://a/bc"}
	regB := LogoutRegistration{OIDCSessionID: "at-b", ClientID: "client-b", UserID: "person:7", BackChannelLogoutURI: "https://b/bc"}
	require.NoError(t, store.Register(ctx, sid, regA))
	require.NoError(t, store.Register(ctx, sid, regB))

	// 只删除其中一条，另一条保留
	require.NoError(t, store.Delete(ctx, sid, "at-a"))

	regs, err := store.ListBySessionID(ctx, sid)
	require.NoError(t, err)
	require.Len(t, regs, 1)
	require.Equal(t, "client-b", regs[0].ClientID)
}

func TestLogoutQueueEnqueueDequeue(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.Background()
	job := LogoutJob{
		SessionID:            "sid-1",
		PersonID:             42,
		OIDCSessionID:        "at-1",
		ClientID:             "client-a",
		UserID:               "person:42",
		BackChannelLogoutURI: "https://a.example.com/bc-logout",
	}
	require.NoError(t, EnqueueLogout(ctx, job))

	got, ok, err := DequeueLogout(ctx, 0)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "client-a", got.ClientID)
	require.Equal(t, "person:42", got.UserID)
	require.Equal(t, "sid-1", got.SessionID)
	require.Equal(t, uint(42), got.PersonID)
}
