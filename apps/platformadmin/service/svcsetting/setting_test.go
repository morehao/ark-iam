package svcsetting

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtosetting"
	"github.com/morehao/ark-iam/apps/platformadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/gsetting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSvc 建 sqlite 并初始化 gsetting（含 core_setting 表），返回服务实例。
func setupSvc(t *testing.T) SettingSvc {
	t.Helper()
	db := testutil.SetupSQLite(t) // gsetting.New 内部会 AutoMigrate core_setting
	if _, err := gsetting.New(db); err != nil {
		t.Fatalf("gsetting.New: %v", err)
	}
	return NewSettingSvc()
}

func TestCreate_GroupWhitelist(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	// 非白名单分组被拒。
	err := svc.Create(ctx, &dtosetting.SettingCreateReq{Group: "evil", Key: "k", Value: "v", ValueType: "string"})
	assert.ErrorIs(t, err, code.GetError(code.SettingGroupNotAllowedError))

	// 白名单分组可创建。
	err = svc.Create(ctx, &dtosetting.SettingCreateReq{Group: "notify", Key: "enabled", Value: "true", ValueType: "bool"})
	assert.NoError(t, err)
}

func TestCreate_EncryptedMaskedOnRead(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	require.NoError(t, svc.Create(ctx, &dtosetting.SettingCreateReq{
		Group: "notify", Key: "mail.password", Value: "s3cr3t", ValueType: "string", Encrypted: true,
	}))

	page, err := svc.PageList(ctx, &dtosetting.SettingPageListReq{Group: "notify"})
	require.NoError(t, err)
	require.Len(t, page.List, 1)
	assert.Equal(t, MaskedValue, page.List[0].Value, "加密项读取应为掩码")

	// 详情同样掩码。
	detail, err := svc.Detail(ctx, &dtosetting.SettingDetailReq{SettingID: page.List[0].ID})
	require.NoError(t, err)
	assert.Equal(t, MaskedValue, detail.Value)
}

func TestUpdate_MaskedValueKeepsSecret(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	require.NoError(t, svc.Create(ctx, &dtosetting.SettingCreateReq{
		Group: "notify", Key: "mail.password", Value: "orig", ValueType: "string", Encrypted: true,
	}))
	page, err := svc.PageList(ctx, &dtosetting.SettingPageListReq{Group: "notify"})
	require.NoError(t, err)
	id := page.List[0].ID

	// 用掩码值更新 -> 不应覆盖原密文。
	require.NoError(t, svc.Update(ctx, &dtosetting.SettingUpdateReq{SettingID: id, Value: MaskedValue, Encrypted: true}))

	// 直接读回（GetByID 解密），密文应仍为 "orig"。
	got, err := gsetting.GetAdmin().GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "orig", got.Value, "掩码值更新不应改变原值")

	// 用新明文更新 -> 应改为 newpwd。
	require.NoError(t, svc.Update(ctx, &dtosetting.SettingUpdateReq{SettingID: id, Value: "newpwd", Encrypted: true}))
	got2, err := gsetting.GetAdmin().GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "newpwd", got2.Value)
}

func TestPageList_RejectsNonWhitelistGroup(t *testing.T) {
	svc := setupSvc(t)
	_, err := svc.PageList(context.Background(), &dtosetting.SettingPageListReq{Group: "evil"})
	assert.ErrorIs(t, err, code.GetError(code.SettingGroupNotAllowedError))
}
