package revoke

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Validation(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
}

func TestRevoke_SendsHintAndAuth(t *testing.T) {
	var gotHint string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHint = r.PostFormValue("token_type_hint")
		assert.Equal(t, "refresh-token-value", r.PostFormValue("token"))
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "c", user)
		assert.Equal(t, "s", pass)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := New(Config{Endpoint: srv.URL, ClientID: "c", ClientSecret: "s"})
	require.NoError(t, err)
	require.NoError(t, client.Revoke(context.Background(), "refresh-token-value", TokenTypeRefreshToken))
	assert.Equal(t, "refresh_token", gotHint)

	require.NoError(t, client.Revoke(context.Background(), "refresh-token-value", ""))
	assert.Equal(t, "", gotHint)
}

func TestRevoke_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client, err := New(Config{Endpoint: srv.URL, ClientID: "c", ClientSecret: "s"})
	require.NoError(t, err)
	require.Error(t, client.Revoke(context.Background(), "", TokenTypeAccessToken))
	require.Error(t, client.Revoke(context.Background(), "t", TokenTypeAccessToken))
}
