package sso

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

func TestHasActiveSession(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.Background()
	store := NewSSOSessionStore()

	// 尚未创建任何会话 -> 无有效会话
	active, err := store.HasActiveSession(ctx, "88")
	require.NoError(t, err)
	require.False(t, active)

	// 创建会话后 -> 有效
	sid, err := store.CreateSession(ctx, "88", []string{"pwd"})
	require.NoError(t, err)
	require.NotEmpty(t, sid)

	active, err = store.HasActiveSession(ctx, "88")
	require.NoError(t, err)
	require.True(t, active)

	// 按 personID 全局撤销后 -> 不再有效
	require.NoError(t, store.RevokeSessionsByPersonID(ctx, "88"))

	active, err = store.HasActiveSession(ctx, "88")
	require.NoError(t, err)
	require.False(t, active)
}

func TestCreateSessionRecordsSessionAudit(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.WithValue(context.Background(), ContextKeyTenantID, "77")
	store := NewSSOSessionStore()

	prev := sessionAuditWriter
	defer func() { sessionAuditWriter = prev }()

	var captured *model.SessionAuditEntity
	sessionAuditWriter = func(_ context.Context, entity *model.SessionAuditEntity) error {
		captured = entity
		return nil
	}

	sid, err := store.CreateSession(ctx, "88", []string{"pwd"})
	require.NoError(t, err)
	require.NotEmpty(t, sid)

	require.NotNil(t, captured, "session audit should be written")
	require.Equal(t, "88", captured.PersonID)
	require.Equal(t, sid, captured.SessionID)
	require.False(t, captured.LoginTime.IsZero())
	require.Equal(t, "77", captured.TenantID, "session audit should carry resolved tenant_id from ctx")

	// 清理共享 Redis 中的测试会话，避免污染依赖干净状态的其它用例
	_ = store.RevokeSessionsByPersonID(ctx, "88")
}

func TestCreateSessionToleratesAuditWriteFailure(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)

	ctx := context.Background()
	store := NewSSOSessionStore()

	prev := sessionAuditWriter
	defer func() { sessionAuditWriter = prev }()

	sessionAuditWriter = func(_ context.Context, _ *model.SessionAuditEntity) error {
		return errors.New("db unavailable")
	}

	// 审计落库失败不得阻断 SSO 会话创建
	sid, err := store.CreateSession(ctx, "99", []string{"pwd"})
	require.NoError(t, err)
	require.NotEmpty(t, sid)

	// 清理共享 Redis 中的测试会话
	_ = store.RevokeSessionsByPersonID(ctx, "99")
}

// fakeSSOStore 用于验证 ValidateTokenSession 的 sid 语义，不触达 Redis。
type fakeSSOStore struct {
	personHasActive bool
	validatePerson  string
}

func (f *fakeSSOStore) CreateSession(ctx context.Context, personID string, amr []string) (string, error) {
	return "", nil
}
func (f *fakeSSOStore) ValidateSession(ctx context.Context, sessionID string) (string, error) {
	return f.validatePerson, nil
}
func (f *fakeSSOStore) SessionAMR(ctx context.Context, sessionID string) []string { return nil }
func (f *fakeSSOStore) SessionAuthTime(ctx context.Context, sessionID string) time.Time {
	return time.Time{}
}
func (f *fakeSSOStore) RevokeSession(ctx context.Context, sessionID string) error { return nil }
func (f *fakeSSOStore) RevokeSessionsByPersonID(ctx context.Context, personID string) error {
	return nil
}
func (f *fakeSSOStore) HasActiveSession(ctx context.Context, personID string) (bool, error) {
	return f.personHasActive, nil
}

// TestValidateTokenSessionRequireSID 覆盖 DG-SESSION-1：无 sid 时
// requireSID=true 必须 fail-closed（不回退 person 活性）；requireSID=false 回退旧兼容。
func TestValidateTokenSessionRequireSID(t *testing.T) {
	ctx := context.Background()
	// person 有活跃会话，但 token 无 sid
	store := &fakeSSOStore{personHasActive: true}

	// D：requireSID=true，无 sid → false（fail-closed）
	ok, err := ValidateTokenSession(ctx, store, "p1", "", true)
	require.NoError(t, err)
	require.False(t, ok, "requireSID 且无 sid 必须 fail-closed，不得回退 person")

	// 平台：requireSID=false，无 sid → 回退 HasActiveSession(true)
	ok, err = ValidateTokenSession(ctx, store, "p1", "", false)
	require.NoError(t, err)
	require.True(t, ok, "非 requireSID 无 sid 应回退 person 活性校验")

	// 有 sid：按成员匹配
	store2 := &fakeSSOStore{validatePerson: "p1"}
	ok, err = ValidateTokenSession(ctx, store2, "p1", "sid", true)
	require.NoError(t, err)
	require.True(t, ok)
	// sid 归属他人 → 拒绝
	ok, err = ValidateTokenSession(ctx, store2, "p2", "sid", true)
	require.NoError(t, err)
	require.False(t, ok)
}
