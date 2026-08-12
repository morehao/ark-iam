package svcoidc

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/stretchr/testify/require"
)

func newTestWorker(t *testing.T) *logoutWorker {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return NewLogoutWorker(key, "test-key", "https://iam.example.com/oidc")
}

func TestBuildLogoutToken(t *testing.T) {
	worker := newTestWorker(t)

	job := sso.LogoutJob{
		SessionID:            "sid-abc",
		PersonID:             42,
		OIDCSessionID:        "at-1",
		ClientID:             "client-a",
		UserID:               "person:42",
		BackChannelLogoutURI: "https://a.example.com/bc-logout",
	}
	token, err := worker.buildLogoutToken(job)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}
