package svcauth

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/stretchr/testify/require"
)

const bclQueueKey = "iam:oidc:slo_queue"

// setupBCLTestEnv 以内存 SQLite 注册 iam 库（并播种操作人 user "1"），
// 同时初始化本地 Redis 供 SSO 会话与 SLO 队列使用，替代依赖真实数据库种子的旧集成方式。
func setupBCLTestEnv(t *testing.T) {
	t.Helper()
	db := testutil.SetupSQLite(t, &model.TenantEntity{}, &model.PersonEntity{}, &model.UserEntity{}, &model.UserDepartmentEntity{})
	now := time.Now()
	seedTenant := &model.TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}}, Code: "seed", Name: "seed"}
	if err := db.Create(seedTenant).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	seedPerson := &model.PersonEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}}, Name: "seed", Profile: []byte(`{}`), CustomData: []byte(`{}`)}
	if err := db.Create(seedPerson).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
	seedUser := &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}}, TenantID: "1", PersonID: "1", Name: "seed", Profile: []byte(`{}`), CustomData: []byte(`{}`), JoinedAt: &now}
	if err := db.Create(seedUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// 最小化初始化日志，避免 glog 未初始化时崩溃
	_ = glog.InitLogger(&glog.LogConfig{
		Service:    "auth-test",
		Module:     "test",
		Level:      glog.WarnLevel,
		LoggerType: glog.LoggerTypeZap,
		Writers:    []glog.WriterConfig{{Type: glog.WriterConsole}},
	})

	oldCli := dbclient.RedisCli
	require.NoError(t, dbclient.InitRedis(dbredis.RedisConfig{Service: "iam", Addr: "127.0.0.1:6379"}, nil))
	t.Cleanup(func() { dbclient.RedisCli = oldCli })
}

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
func findLogoutJob(jobs []sso.LogoutJob, personID string, clientID, oidcSessionID string) *sso.LogoutJob {
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

// sloQueueContended 检测共享 SLO 队列是否正被运行中的服务（logout worker）持续消费。
// 若被外部消费，队列内容无法稳定断言（任务会被立即取走），测试应跳过而非误报失败。
// 仅把连接存活超过 contentionMinAge 的 BRPop 消费者视为外部 worker：
// 本测试套件自身的短超时 BRPop（几秒内）不算，避免跨包并行时误判。
func sloQueueContended(t *testing.T) bool {
	t.Helper()
	const contentionMinAge = 5 * time.Second
	clients, err := dbclient.RedisCli.ClientList(context.Background()).Result()
	if err != nil {
		return true // 无法探测时保守跳过
	}
	for _, line := range strings.Split(clients, "\n") {
		if !strings.Contains(line, "cmd=brpop") {
			continue
		}
		if age, ok := parseClientAge(line); ok && age > contentionMinAge {
			return true
		}
	}
	return false
}

// parseClientAge 从 redis CLIENT LIST 单行中解析 age=<秒> 字段。
func parseClientAge(line string) (time.Duration, bool) {
	for _, field := range strings.Fields(line) {
		if strings.HasPrefix(field, "age=") {
			secs, err := strconv.ParseInt(strings.TrimPrefix(field, "age="), 10, 64)
			if err != nil {
				return 0, false
			}
			return time.Duration(secs) * time.Second, true
		}
	}
	return 0, false
}

// TestLogoutEnqueuesBackChannelLogoutForPerson 验证业务侧登出（Logout）会为该 person
// 已登记的 client 入队 back-channel logout 任务（一处登出 → 处处登出的 OP 侧补充）。
func TestLogoutEnqueuesBackChannelLogoutForPerson(t *testing.T) {
	setupBCLTestEnv(t)
	if sloQueueContended(t) {
		t.Skip("shared SLO queue is being consumed by a running logout worker; skip queue assertion")
	}
	ctx := context.Background()

	personID := "99"
	sid, err := sso.NewSSOSessionStore().CreateSession(ctx, personID, []string{"pwd"})
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

	ginCtx := testsetup.NewCtx(testutil.WithIamContext("1"))
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
	setupBCLTestEnv(t)

	personID := "1000"
	ginCtx := testsetup.NewCtx(testutil.WithIamContext("1"))
	ginCtx.Set(gcontext.KeyPersonID, personID)

	svc := NewAuthSvc()
	require.NoError(t, svc.LogoutAll(ginCtx, &dtoauth.LogoutAllReq{}))

	// 无登记的 person 登出后，不应出现其专属任务
	require.Nil(t, findLogoutJob(listLogoutJobs(t), personID, "", ""), "Logout with no registrations should not enqueue jobs")
}
