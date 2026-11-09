package rp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenClient_Validation(t *testing.T) {
	_, err := NewTokenClient(ClientCredentialsConfig{}, nil)
	require.Error(t, err)
	_, err = NewTokenClient(ClientCredentialsConfig{TokenURL: "http://x"}, nil)
	require.Error(t, err)
	_, err = NewTokenClient(ClientCredentialsConfig{TokenURL: "http://x", ClientID: "c"}, nil)
	require.Error(t, err)
}

func TestTokenClient_CachesAndRenews(t *testing.T) {
	var requests int32
	expiresIn := 3600
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		require.Equal(t, "client_credentials", r.PostFormValue("grant_type"))
		require.Equal(t, "directory.read", r.PostFormValue("scope"))
		require.Equal(t, "http://backend", r.PostFormValue("resource"))
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "svc", user)
		assert.Equal(t, "secret", pass)
		_, _ = fmt.Fprintf(w, `{"access_token":"tok-%d","token_type":"Bearer","expires_in":%d}`,
			atomic.LoadInt32(&requests), expiresIn)
	}))
	defer srv.Close()

	client, err := NewTokenClient(ClientCredentialsConfig{
		TokenURL:     srv.URL,
		ClientID:     "svc",
		ClientSecret: "secret",
		Scopes:       []string{"directory.read"},
		Resource:     "http://backend",
	}, nil)
	require.NoError(t, err)

	first, err := client.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "tok-1", first)
	second, err := client.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, first, second, "有效期内必须复用缓存令牌")
	assert.Equal(t, int32(1), atomic.LoadInt32(&requests))

	// 令牌即将过期（<60s 余量）时必须提前续期。
	client.mu.Lock()
	client.expiresAt = time.Now().Add(10 * time.Second)
	client.mu.Unlock()
	third, err := client.Token(context.Background())
	require.NoError(t, err)
	assert.NotEqual(t, first, third)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requests))

	client.Invalidate()
	_, err = client.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requests))
}

func TestTokenClient_ErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer srv.Close()

	client, err := NewTokenClient(ClientCredentialsConfig{
		TokenURL: srv.URL, ClientID: "c", ClientSecret: "s",
	}, nil)
	require.NoError(t, err)
	_, err = client.Token(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestTokenClient_EmptyAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	client, err := NewTokenClient(ClientCredentialsConfig{
		TokenURL: srv.URL, ClientID: "c", ClientSecret: "s",
	}, nil)
	require.NoError(t, err)
	_, err = client.Token(context.Background())
	assert.Contains(t, err.Error(), "no access_token")
}
