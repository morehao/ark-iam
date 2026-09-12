package sso

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

func TestLogoutRegistrationRegisterAndListBySessionID(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.Background()
	store := NewSLOStore()
	sid := fmt.Sprintf("test-slo-session-1-%d", time.Now().UnixNano())

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
	sid := fmt.Sprintf("test-slo-session-2-%d", time.Now().UnixNano())

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
	// 使用测试独立的队列键，避免与运行中的 logout worker 共享消费（共享开发 Redis 场景）。
	queueKey := fmt.Sprintf("iam:oidc:slo_queue:test:%d", time.Now().UnixNano())
	job := LogoutJob{
		SessionID:            "sid-1",
		PersonID:             "42",
		OIDCSessionID:        "at-1",
		ClientID:             "client-a",
		UserID:               "person:42",
		BackChannelLogoutURI: "https://a.example.com/bc-logout",
	}
	require.NoError(t, enqueueLogout(ctx, queueKey, job))

	got, ok, err := dequeueLogout(ctx, queueKey, 5*time.Second)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "client-a", got.ClientID)
	require.Equal(t, "person:42", got.UserID)
	require.Equal(t, "sid-1", got.SessionID)
	require.Equal(t, "42", got.PersonID)
}

// TestEnqueueLogoutsByPersonID 验证「按自然人批量投递背信道通知」：
// 该自然人跨全部 SSO 会话登记的 client 都要入队（登出/租户挂起都依赖它），
// 返回值为实际入队数量，空 personID 不入队任何任务。
func TestEnqueueLogoutsByPersonID(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.Background()
	store := NewSLOStore()
	// 独立队列键，避免与运行中的 logout worker 争抢消费。
	queueKey := fmt.Sprintf("iam:oidc:slo_queue:test:%d", time.Now().UnixNano())
	personID := fmt.Sprintf("bcl-person-%d", time.Now().UnixNano())

	// 空 personID：不入队，且不触碰 Redis
	count, err := enqueueLogoutsByPersonID(ctx, queueKey, "")
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// 无登记：入队数为 0
	count, err = enqueueLogoutsByPersonID(ctx, queueKey, personID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// 同一自然人的两个 SSO 会话，各登记一个 client
	sidOne, err := NewSSOSessionStore().CreateSession(ctx, personID, []string{"pwd"})
	require.NoError(t, err)
	defer func() { _ = NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID) }()
	sidTwo, err := NewSSOSessionStore().CreateSession(ctx, personID, []string{"pwd"})
	require.NoError(t, err)

	require.NoError(t, store.Register(ctx, sidOne, LogoutRegistration{
		OIDCSessionID: "at-one", ClientID: "client-one",
		UserID: "person:" + personID, BackChannelLogoutURI: "https://one.example.com/bc",
	}))
	require.NoError(t, store.Register(ctx, sidTwo, LogoutRegistration{
		OIDCSessionID: "at-two", ClientID: "client-two",
		UserID: "person:" + personID, BackChannelLogoutURI: "https://two.example.com/bc",
	}))

	count, err = enqueueLogoutsByPersonID(ctx, queueKey, personID)
	require.NoError(t, err)
	require.Equal(t, 2, count, "两个会话的登记都应入队")

	got := make(map[string]LogoutJob)
	for i := 0; i < 2; i++ {
		job, ok, err := dequeueLogout(ctx, queueKey, 5*time.Second)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, personID, job.PersonID)
		require.False(t, job.CreatedAt.IsZero(), "任务必须带入队时间戳")
		got[job.ClientID] = job
	}
	require.Equal(t, "at-one", got["client-one"].OIDCSessionID)
	require.Equal(t, "https://one.example.com/bc", got["client-one"].BackChannelLogoutURI)
	require.Equal(t, "at-two", got["client-two"].OIDCSessionID)
	require.Equal(t, "https://two.example.com/bc", got["client-two"].BackChannelLogoutURI)
}
