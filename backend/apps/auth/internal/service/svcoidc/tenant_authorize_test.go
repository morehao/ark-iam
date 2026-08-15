package svcoidc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func TestTenantHintMiddleware_injectsRequestContext(t *testing.T) {
	r := gin.New()
	r.Any("/authorize", tenantHintMiddleware(), func(ctx *gin.Context) {
		v, ok := ctx.Request.Context().Value(ctxTenantHintKey).(string)
		assert.True(t, ok, "expected tenant hint in request context")
		assert.Equal(t, "5", v)
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?tenant=5", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantHintMiddleware_noTenant(t *testing.T) {
	r := gin.New()
	r.Any("/authorize", tenantHintMiddleware(), func(ctx *gin.Context) {
		assert.Nil(t, ctx.Request.Context().Value(ctxTenantHintKey))
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantHintMiddleware_passesAnyNonEmptyHint(t *testing.T) {
	// string-id 改造后 tenant hint 是 UUID 字符串：非空即透传，合法性由下游成员租户校验兜底。
	r := gin.New()
	r.Any("/authorize", tenantHintMiddleware(), func(ctx *gin.Context) {
		v, ok := ctx.Request.Context().Value(ctxTenantHintKey).(string)
		assert.True(t, ok, "expected tenant hint in request context")
		assert.Equal(t, "abc", v)
		ctx.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?tenant=abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

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

	ctx := context.WithValue(context.Background(), ctxTenantHintKey, "5")
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

	withValue := context.WithValue(context.Background(), ctxTenantHintKey, "7")
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
