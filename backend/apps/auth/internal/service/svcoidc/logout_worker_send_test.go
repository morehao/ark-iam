package svcoidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

// decodeJWTClaims 解码 JWT 的 payload，不验签（测试用）。
func decodeJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims
}

func TestLogoutWorkerSendsBackChannelLogoutAndDeletesRegistration(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	ctx := context.Background()

	var receivedToken string
	done := make(chan struct{}, 1)
	rpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析 form 体中的 logout_token
		_ = r.ParseForm()
		receivedToken = r.FormValue("logout_token")
		done <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer rpServer.Close()

	// 登记一条待通知记录
	sid := "slo-e2e-sid-1"
	reg := sso.LogoutRegistration{
		OIDCSessionID:        "at-e2e",
		ClientID:             "client-e2e",
		UserID:               "person:88",
		BackChannelLogoutURI: rpServer.URL,
	}
	require.NoError(t, sso.NewSLOStore().Register(ctx, sid, reg))

	worker := newTestWorker(t)

	// 触发发送（走 handleJob：成功后删除登记）
	worker.handleJob(ctx, sso.LogoutJob{
		SessionID:            sid,
		PersonID:             "88",
		OIDCSessionID:        "at-e2e",
		ClientID:             "client-e2e",
		UserID:               "person:88",
		BackChannelLogoutURI: rpServer.URL,
	})

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("RP did not receive logout request")
	}

	require.NotEmpty(t, receivedToken, "RP should receive a logout_token")
	claims := decodeJWTClaims(t, receivedToken)

	// 必须含 events.backchannel-logout（标识 logout token）
	events, ok := claims["events"].(map[string]any)
	require.True(t, ok, "logout token must contain events claim")
	_, ok = events["http://schemas.openid.net/event/backchannel-logout"]
	require.True(t, ok, "events must contain backchannel-logout")

	// aud 必须是 clientID
	aud, ok := claims["aud"].(string)
	if !ok {
		if audList, isList := claims["aud"].([]any); isList {
			require.NotEmpty(t, audList)
			aud = audList[0].(string)
		}
	}
	require.Equal(t, "client-e2e", aud)

	// 成功后登记应被删除
	regs, err := sso.NewSLOStore().ListBySessionID(context.Background(), sid)
	require.NoError(t, err)
	require.Empty(t, regs, "registration should be deleted after successful send")
}
