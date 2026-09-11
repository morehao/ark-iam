package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	_ "github.com/morehao/golib/glog/driver/zap" // 注册 zap 日志驱动，dbredis 初始化依赖已初始化的 glog 配置
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// bclQueueKey 背信道通知队列（与 sso 包内的生产队列键一致）。
const bclQueueKey = "iam:oidc:slo_queue"

// setupTenantTestEnv 以内存 SQLite 注册 iam 库，并在独立 Redis 逻辑库（DB 5）上初始化连接。
// pkg 模块无法复用 apps 的 testutil（pkg 不得依赖 apps），故直接注册测试库。
// 选独立逻辑库是为了与开发环境可能正在运行的 logout worker 隔离：worker 只消费 DB 0 的队列，
// 这样本测试对 SLO 队列的断言是确定性的，不需要「队列被消费则跳过」的兜底。
func setupTenantTestEnv(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:tenant_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.UserEntity{},
		&model.RefreshTokenEntity{},
		&model.SessionAuditEntity{},
	))
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})

	_ = glog.InitLogger(&glog.LogConfig{
		Service:    "tenant-test",
		Module:     "test",
		Level:      glog.WarnLevel,
		LoggerType: glog.LoggerTypeZap,
		Writers:    []glog.WriterConfig{{Type: glog.WriterConsole}},
	})
	oldCli := dbclient.RedisCli
	require.NoError(t, dbclient.InitRedis(dbredis.RedisConfig{Service: "iam", Addr: "127.0.0.1:6379", DB: 5}, nil))
	t.Cleanup(func() { dbclient.RedisCli = oldCli })

	return db
}

func seedTenantUser(t *testing.T, db *gorm.DB, id, tenantID, personID string) {
	t.Helper()
	now := time.Now()
	entity := &model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		PersonID:   personID,
		Name:       "user-" + id,
		Profile:    []byte(`{}`),
		CustomData: []byte(`{}`),
		JoinedAt:   &now,
	}
	require.NoError(t, db.Create(entity).Error)
}

func seedRefreshToken(t *testing.T, db *gorm.DB, id, personID string) {
	t.Helper()
	entity := &model.RefreshTokenEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		PersonID:   personID,
		TenantID:   "t1",
		Token:      "hash-" + id,
	}
	require.NoError(t, db.Create(entity).Error)
}

// TestRevokeMemberSessions 验证租户挂起时的会话切断语义：
// 只撤销该租户成员的 refresh token 与 SSO 会话（同一自然人多个用户去重），不波及其它租户成员，
// 并向该自然人名下已登记的应用投递 back-channel logout 通知。
func TestRevokeMemberSessions(t *testing.T) {
	db := setupTenantTestEnv(t)
	ctx := context.Background()

	suffix := time.Now().UnixNano()
	memberPerson := fmt.Sprintf("p-member-%d", suffix)
	otherPerson := fmt.Sprintf("p-other-%d", suffix)

	// 同一自然人在租户内有两个用户（去重场景）+ 另一租户的一个成员
	seedTenantUser(t, db, fmt.Sprintf("u1-%d", suffix), "t1", memberPerson)
	seedTenantUser(t, db, fmt.Sprintf("u2-%d", suffix), "t1", memberPerson)
	seedTenantUser(t, db, fmt.Sprintf("u3-%d", suffix), "t2", otherPerson)

	seedRefreshToken(t, db, fmt.Sprintf("rt1-%d", suffix), memberPerson)
	seedRefreshToken(t, db, fmt.Sprintf("rt2-%d", suffix), otherPerson)

	// 两个自然人各有一个活动的 SSO 会话，成员的那个登记了一个待通知的 client
	ssoStore := sso.NewSSOSessionStore()
	memberSID, err := ssoStore.CreateSession(ctx, memberPerson, []string{"pwd"})
	require.NoError(t, err)
	otherSID, err := ssoStore.CreateSession(ctx, otherPerson, []string{"pwd"})
	require.NoError(t, err)
	require.NotEmpty(t, otherSID)
	t.Cleanup(func() {
		_ = ssoStore.RevokeSessionsByPersonID(ctx, memberPerson)
		_ = ssoStore.RevokeSessionsByPersonID(ctx, otherPerson)
	})

	clientID := fmt.Sprintf("client-%d", suffix)
	require.NoError(t, sso.NewSLOStore().Register(ctx, memberSID, sso.LogoutRegistration{
		OIDCSessionID:        "at-member",
		ClientID:             clientID,
		UserID:               "person:" + memberPerson,
		BackChannelLogoutURI: "https://member.example.com/bc",
	}))

	require.NoError(t, RevokeMemberSessions(ctx, "t1"))

	// 本租户成员的 SSO 会话被撤销，其它租户成员不受影响
	memberActive, err := ssoStore.HasActiveSession(ctx, memberPerson)
	require.NoError(t, err)
	require.False(t, memberActive, "租户成员的 SSO 会话应被撤销")
	otherActive, err := ssoStore.HasActiveSession(ctx, otherPerson)
	require.NoError(t, err)
	require.True(t, otherActive, "其它租户成员的 SSO 会话不应被撤销")

	// refresh token：本租户成员被撤销，其它租户成员保持有效
	require.True(t, refreshTokenRevoked(t, db, "rt1", suffix), "租户成员的 refresh token 应被撤销")
	require.False(t, refreshTokenRevoked(t, db, "rt2", suffix), "其它租户成员的 refresh token 不应被撤销")

	// back-channel 通知：该成员登记的应用应收到 logout 任务（独立 Redis 逻辑库，断言确定性）
	job := findTenantLogoutJob(t, memberPerson, clientID)
	require.NotNil(t, job, "挂起租户成员时应投递 back-channel logout 通知")
	require.Equal(t, "at-member", job.OIDCSessionID)
	require.Equal(t, "https://member.example.com/bc", job.BackChannelLogoutURI)
	require.False(t, job.CreatedAt.IsZero(), "任务必须带入队时间戳")
	removeTenantLogoutJob(t, *job)
}

// TestRevokeMemberSessionsEmptyTenantID 空租户ID不做任何事（幂等安全）。
func TestRevokeMemberSessionsEmptyTenantID(t *testing.T) {
	setupTenantTestEnv(t)
	require.NoError(t, RevokeMemberSessions(context.Background(), ""))
}

func refreshTokenRevoked(t *testing.T, db *gorm.DB, idPrefix string, suffix int64) bool {
	t.Helper()
	var entity model.RefreshTokenEntity
	require.NoError(t, db.Where("id = ?", fmt.Sprintf("%s-%d", idPrefix, suffix)).First(&entity).Error)
	return entity.RevokedAt != nil
}

func findTenantLogoutJob(t *testing.T, personID, clientID string) *sso.LogoutJob {
	t.Helper()
	vals, err := dbclient.RedisCli.LRange(context.Background(), bclQueueKey, 0, -1).Result()
	require.NoError(t, err)
	for _, v := range vals {
		var job sso.LogoutJob
		if json.Unmarshal([]byte(v), &job) != nil {
			continue
		}
		if job.PersonID == personID && job.ClientID == clientID {
			return &job
		}
	}
	return nil
}

func removeTenantLogoutJob(t *testing.T, job sso.LogoutJob) {
	t.Helper()
	raw, err := json.Marshal(job)
	require.NoError(t, err)
	_ = dbclient.RedisCli.LRem(context.Background(), bclQueueKey, 1, string(raw)).Err()
}
