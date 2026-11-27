package appassembly

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/morehao/ark-iam/pkg/dbclient"
)

// fakeSSOStore 是 SSOSessionStore 的最小桩：只实现 HasActiveSession，其余 panic（本测试用不到）。
type fakeSSOStore struct {
	personID string
	active   bool
	err      error
}

func (f *fakeSSOStore) HasActiveSession(context.Context, string) (bool, error) {
	return f.active, f.err
}
func (f *fakeSSOStore) CreateSession(context.Context, string, []string) (string, error) {
	panic("unused") //nolint
}
func (f *fakeSSOStore) ValidateSession(context.Context, string) (string, error) {
	return f.personID, f.err
}
func (f *fakeSSOStore) SessionAMR(context.Context, string) []string { panic("unused") } //nolint
func (f *fakeSSOStore) SessionAuthTime(context.Context, string) time.Time {
	panic("unused") //nolint
}
func (f *fakeSSOStore) RevokeSession(context.Context, string) error            { panic("unused") } //nolint
func (f *fakeSSOStore) RevokeSessionsByPersonID(context.Context, string) error { panic("unused") } //nolint

// withRedisClient 临时设置全局 dbclient.RedisCli，测试后还原。
func withRedisClient(t *testing.T, cli *redis.Client) {
	t.Helper()
	prev := dbclient.RedisCli
	dbclient.RedisCli = cli
	t.Cleanup(func() { dbclient.RedisCli = prev })
}

// TestValidateSSOSession_FailClosedOnError 回归 P1-7：
// Redis 已配置但会话查询出错时，必须 fail-closed（返回 false），而不是放行已撤销的会话。
func TestValidateSSOSession_FailClosedOnError(t *testing.T) {
	// 指向一个不可达地址，仅用于让 RedisCli 非 nil；HasActiveSession 由桩返回错误，不真的连它。
	withRedisClient(t, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))

	store := &fakeSSOStore{err: errors.New("redis down")}
	if got := validateSSOSession(context.Background(), store, "person:1", "", false, ""); got {
		t.Fatal("SSO validation must fail-closed (false) when Redis errors")
	}
}

// TestValidateSSOSession_NoRedisAllows 未配置 Redis（nil）时放行（由令牌 TTL 兜底）。
func TestValidateSSOSession_NoRedisAllows(t *testing.T) {
	withRedisClient(t, nil)
	store := &fakeSSOStore{active: false}
	if got := validateSSOSession(context.Background(), store, "person:1", "", false, ""); !got {
		t.Fatal("no Redis configured must allow (rely on access token TTL)")
	}
}

// TestValidateSSOSession_MachineTokenBypasses 机器凭证不依赖 SSO 会话，直接放行（即使 store 报错）。
func TestValidateSSOSession_MachineTokenBypasses(t *testing.T) {
	withRedisClient(t, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))
	store := &fakeSSOStore{err: errors.New("redis down")}
	if got := validateSSOSession(context.Background(), store, "svc", "", true, ""); !got {
		t.Fatal("machine token must bypass SSO session validation")
	}
}

// TestValidateSSOSession_ActiveAndRevoked 正常路径：会话有效放行、无效拒绝。
func TestValidateSSOSession_ActiveAndRevoked(t *testing.T) {
	withRedisClient(t, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))
	if got := validateSSOSession(context.Background(), &fakeSSOStore{active: true}, "p", "", false, ""); !got {
		t.Fatal("active session must pass")
	}
	if got := validateSSOSession(context.Background(), &fakeSSOStore{active: false}, "p", "", false, ""); got {
		t.Fatal("revoked session must be rejected")
	}
}

func TestValidateSSOSessionUsesTokenSession(t *testing.T) {
	withRedisClient(t, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))
	store := &fakeSSOStore{active: true, err: errors.New("old session revoked")}
	if validateSSOSession(t.Context(), store, "person-one", "old-sid", false, "") {
		t.Fatal("another active session must not reactivate the revoked token session")
	}
	store.err = nil
	store.personID = "person-one"
	if !validateSSOSession(t.Context(), store, "person-one", "new-sid", false, "") {
		t.Fatal("the current session should pass")
	}
}
