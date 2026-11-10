package oidcop

import (
	"context"
	"testing"
	"time"

	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/require"
)

// TestProtocolStoreDeclaresTenantScope 锁定协议层（`/oidc/*`）的作用域声明不退化。
//
// 协议端点没有租户作用域中间件：op.Storage 拿到的就是「零作用域」的纯 ctx。
// 测试库与生产共用同一份 fail-closed 插件（testutil.SetupSQLite → dbclient.RegisterDBForTest），
// 因此协议层若漏声明作用域，这里会以 ErrTenantScopeMissing 失败，
// 而不是在线上退化成"静默跨租户读"或"静默 0 行 UPDATE"。
//
// 覆盖两类真实形态：
//  1. 按全局唯一键反查（client_id / API Key 摘要）→ 跨全部租户；
//  2. 按自然人列出成员关系（私有 claim 的 tenant_id）→ 跨全部租户。
//
// 注：原第 3 类「按行所属租户写回 last_used_at」的用例随 `api_key.last_used_at` 下线一并删除
// （P4：该列与降频写回块已整体移除，协议层不再有行级写回路径）。
func TestProtocolStoreDeclaresTenantScope(t *testing.T) {
	db := testutil.SetupSQLite(t,
		&model.ApplicationClientEntity{},
		&model.ApiKeyEntity{},
		&model.UserEntity{},
	)

	rawAPIKey := "probe-raw-api-key"
	probeNow := time.Now()
	require.NoError(t, db.Create(&model.ApplicationClientEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ac-b"}},
		TenantID:   "t-b",
		AppID:      "app-b",
		Code:       "probe_client_b",
		Name:       "probe-b",
		Status:     model.ApplicationClientStatusEnable,
	}).Error)
	require.NoError(t, db.Create(&model.ApiKeyEntity{
		BaseEntity:  gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ak-a"}},
		TenantID:    "t-a",
		OwnerUserID: "u-a",
		Name:        "probe-key",
		KeyHash:     credential.HashSecret(rawAPIKey),
		KeyPrefix:   "probe-r",
	}).Error)
	require.NoError(t, db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "u-a"}},
		TenantID:   "t-a",
		PersonID:   "p-probe",
		UserType:   model.UserTypeMember,
		JoinedAt:   &probeNow,
	}).Error)

	// 前置断言：插件在测试库里确实生效——零作用域的裸查询必须 fail-closed。
	// 没有这一步，下面的"直传纯 ctx 也能查到"可能只是因为插件没挂上（假通过）。
	var probe model.ApplicationClientEntity
	require.ErrorIs(t,
		db.WithContext(context.Background()).Where("code = ?", "probe_client_b").First(&probe).Error,
		dbclient.ErrTenantScopeMissing)

	store := NewPersistentStore()
	protocolCtx := context.Background() // 协议端点真实入参：不含任何租户作用域

	// 1. 全局唯一键：client_id 反查（authorize/token/end_session 必经）
	client, err := store.GetClientByClientID(protocolCtx, "probe_client_b")
	require.NoError(t, err)
	require.Equal(t, "probe_client_b", client.GetID())

	// 1'. 全局唯一键：API Key 摘要反查（api key 客户端通道必经）
	apiKey, err := store.LookupApiKeyByRawKey(protocolCtx, rawAPIKey)
	require.NoError(t, err)
	require.NotNil(t, apiKey)
	require.Equal(t, "t-a", apiKey.TenantID)

	// 2. 自然人范围：私有 claim 的 tenant_id（多租户列表在协议层可见）
	claims, err := store.GetPrivateClaimsFromScopes(protocolCtx, BuildSubject("p-probe"), "probe_client_b", []string{model.ScopeOpenID})
	require.NoError(t, err)
	require.Equal(t, "t-a", claims["tenant_id"])
}
