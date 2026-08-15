package sso

import (
	"context"
	"errors"
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/model"
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
	require.Equal(t, sessionAuditStatusActive, captured.Status)
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
