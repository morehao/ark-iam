package svcoidc

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

func TestHasActiveSession(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameIam)
	defer testsetup.Done(testsetup.AppNameIam)

	ctx := context.Background()
	store := NewSSOSessionStore()

	// 尚未创建任何会话 -> 无有效会话
	active, err := store.HasActiveSession(ctx, 88)
	require.NoError(t, err)
	require.False(t, active)

	// 创建会话后 -> 有效
	sid, err := store.CreateSession(ctx, 88)
	require.NoError(t, err)
	require.NotEmpty(t, sid)

	active, err = store.HasActiveSession(ctx, 88)
	require.NoError(t, err)
	require.True(t, active)

	// 按 personID 全局撤销后 -> 不再有效
	require.NoError(t, store.RevokeSessionsByPersonID(ctx, 88))

	active, err = store.HasActiveSession(ctx, 88)
	require.NoError(t, err)
	require.False(t, active)
}
