package introspect

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

func TestIntrospect_ActiveAndInactive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "svc", user)
		assert.Equal(t, "secret", pass)
		switch r.PostFormValue("token") {
		case "good":
			_, _ = w.Write([]byte(`{"active":true,"scope":"openid","client_id":"c1","sub":"person:1","exp":1999999999,"tenant_id":"t1","user_id":"u1","token_usage":"machine"}`))
		default:
			_, _ = w.Write([]byte(`{"active":false}`))
		}
	}))
	defer srv.Close()

	client, err := New(Config{Endpoint: srv.URL, ClientID: "svc", ClientSecret: "secret"})
	require.NoError(t, err)

	got, err := client.Introspect(context.Background(), "good")
	require.NoError(t, err)
	assert.True(t, got.Active)
	assert.Equal(t, "person:1", got.Subject)
	assert.Equal(t, "u1", got.Claims["user_id"])
	assert.Equal(t, int64(1999999999), got.ExpiresAt)

	got, err = client.Introspect(context.Background(), "revoked")
	require.NoError(t, err)
	assert.False(t, got.Active)
}

func TestIntrospect_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client, err := New(Config{Endpoint: srv.URL, ClientID: "c", ClientSecret: "s"})
	require.NoError(t, err)
	_, err = client.Introspect(context.Background(), "t")
	require.Error(t, err)
}
