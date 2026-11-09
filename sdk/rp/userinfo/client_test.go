package userinfo

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

func TestGet_ParsesStandardAndGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"sub":"person:1","name":"张三","preferred_username":"zhangsan","email":"z@example.com","email_verified":true,"phone_number":"138","picture":"http://img","groups":["admin","ops"],"custom":"x"}`))
	}))
	defer srv.Close()

	client, err := New(Config{Endpoint: srv.URL})
	require.NoError(t, err)
	info, err := client.Get(context.Background(), "tok")
	require.NoError(t, err)
	assert.Equal(t, "person:1", info.Subject)
	assert.Equal(t, "张三", info.DisplayName())
	assert.Equal(t, []string{"admin", "ops"}, info.Groups)
	assert.True(t, info.EmailVerified)
	assert.Equal(t, "x", info.Extra["custom"], "未知声明必须保留而不报错（前向兼容）")
}

func TestDisplayName_Fallback(t *testing.T) {
	assert.Equal(t, "zhangsan", (&Info{PreferredUsername: "zhangsan"}).DisplayName())
	assert.Equal(t, "", (*Info)(nil).DisplayName())
}

func TestGet_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client, err := New(Config{Endpoint: srv.URL})
	require.NoError(t, err)
	_, err = client.Get(context.Background(), "tok")
	require.Error(t, err)
	_, err = client.Get(context.Background(), "")
	require.Error(t, err)
}
