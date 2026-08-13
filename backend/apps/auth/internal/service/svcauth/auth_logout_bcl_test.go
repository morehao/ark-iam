package svcauth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/stretchr/testify/require"
)

const bclQueueKey = "iam:oidc:slo_queue"

// listLogoutJobs 返回调度队列中的全部 job（不消费），供断言使用。
func listLogoutJobs(t *testing.T) []sso.LogoutJob {
	t.Helper()
	vals, err := dbclient.RedisCli.LRange(context.Background(), bclQueueKey, 0, -1).Result()
	require.NoError(t, err)
	jobs := make([]sso.LogoutJob, 0, len(vals))
	for _, v := range vals {
		var job sso.LogoutJob
		if json.Unmarshal([]byte(v), &job) == nil {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// findLogoutJob 在队列中查找目标 job（按 person + client + oidcSession 精确定位）。
func findLogoutJob(jobs []sso.LogoutJob, personID uint, clientID, oidcSessionID string) *sso.LogoutJob {
	for i := range jobs {
		if jobs[i].PersonID == personID && jobs[i].ClientID == clientID && jobs[i].OIDCSessionID == oidcSessionID {
			return &jobs[i]
		}
	}
	return nil
}

// removeLogoutJob 精确移除队列中指定 job（通过 json.Marshal 还原其序列化串），不影响其它任务。
func removeLogoutJob(t *testing.T, job sso.LogoutJob) {
	t.Helper()
	raw, err := json.Marshal(job)
	require.NoError(t, err)
	_ = dbclient.RedisCli.LRem(context.Background(), bclQueueKey, 1, string(raw)).Err()
}

// TestLogoutEnqueuesBackChannelLogoutForPerson 验证业务侧登出（Logout）会为该 person
// 已登记的 client 入队 back-channel logout 任务（一处登出 → 处处登出的 OP 侧补充）。
func TestLogoutEnqueuesBackChannelLogoutForPerson(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	ctx := context.Background()

	personID := uint(99)
	sid, err := sso.NewSSOSessionStore().CreateSession(ctx, personID)
	require.NoError(t, err)
	defer func() {
		_ = sso.NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID)
	}()

	oidcSessionID := "at-bcl-test"
	reg := sso.LogoutRegistration{
		OIDCSessionID:        oidcSessionID,
		ClientID:             "client-bcl",
		UserID:               "person:99",
		BackChannelLogoutURI: "http://rp.example.com/backchannel",
	}
	require.NoError(t, sso.NewSLOStore().Register(ctx, sid, reg))
	defer func() { _ = sso.NewSLOStore().Delete(ctx, sid, oidcSessionID) }()

	ginCtx := testsetup.NewCtx(testutil.WithIamContext(1))
	ginCtx.Set(gcontext.KeyPersonID, personID)

	svc := NewAuthSvc()
	require.NoError(t, svc.Logout(ginCtx, &dtoauth.LogoutReq{RefreshToken: "rt-bcl"}))

	// Logout 使该 person 的登记被入队；轮询等待写入稳定
	var job *sso.LogoutJob
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if job = findLogoutJob(listLogoutJobs(t), personID, "client-bcl", oidcSessionID); job != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NotNil(t, job, "expected a back-channel logout job for person %d", personID)
	require.Equal(t, "person:99", job.UserID)
	require.Equal(t, "http://rp.example.com/backchannel", job.BackChannelLogoutURI)
	require.False(t, job.CreatedAt.IsZero(), "job must carry an enqueue timestamp")

	// 清理本次产生的任务，保持队列干净，避免影响其它测试
	removeLogoutJob(t, *job)
}

// TestLogoutEnqueuesNothingWithoutRegistrations 验证无登记的 person 登出时不会入队垃圾任务。
func TestLogoutEnqueuesNothingWithoutRegistrations(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	personID := uint(1000)
	ginCtx := testsetup.NewCtx(testutil.WithIamContext(1))
	ginCtx.Set(gcontext.KeyPersonID, personID)

	svc := NewAuthSvc()
	require.NoError(t, svc.LogoutAll(ginCtx, &dtoauth.LogoutAllReq{}))

	// 无登记的 person 登出后，不应出现其专属任务
	require.Nil(t, findLogoutJob(listLogoutJobs(t), personID, "", ""), "Logout with no registrations should not enqueue jobs")
}
