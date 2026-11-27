package svcsetting

import (
	"context"
	"fmt"
	"testing"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtosetting"
	"github.com/morehao/ark-iam/apps/platformadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/runtimesetting"
	"github.com/morehao/golib/gsetting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSvc 建 sqlite 并初始化 gsetting（含 core_setting 表），返回服务实例。
func setupSvc(t *testing.T) SettingSvc {
	t.Helper()
	db := testutil.SetupSQLite(t, &gsetting.SettingEntity{})
	store, err := runtimesetting.New(db)
	if err != nil {
		t.Fatal(err)
	}
	runtimesetting.SetProcessStore(store)
	t.Cleanup(func() { runtimesetting.SetProcessStore(nil) })
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
	got, err := processAdmin().GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "orig", got.Value, "掩码值更新不应改变原值")

	// 用新明文更新 -> 应改为 newpwd。
	require.NoError(t, svc.Update(ctx, &dtosetting.SettingUpdateReq{SettingID: id, Value: "newpwd", Encrypted: true}))
	got2, err := processAdmin().GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "newpwd", got2.Value)
}

func TestPageList_RejectsNonWhitelistGroup(t *testing.T) {
	svc := setupSvc(t)
	_, err := svc.PageList(context.Background(), &dtosetting.SettingPageListReq{Group: "evil"})
	assert.ErrorIs(t, err, code.GetError(code.SettingGroupNotAllowedError))
}

func TestSettingGroups_ReturnsWhitelistMeta(t *testing.T) {
	svc := setupSvc(t)
	resp, err := svc.Groups(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, resp.List)

	// 白名单分组成员、且每个都带中文说明（消除前端手抄 schema 的关键）。
	got := map[string]string{}
	for _, g := range resp.List {
		got[g.Group] = g.Description
	}
	for _, g := range AllowedGroups {
		_, ok := got[g]
		assert.Truef(t, ok, "分组 %s 应出现在 Groups() 中", g)
	}
	assert.NotEmpty(t, got["notify"])
	assert.NotEmpty(t, got["security.login"])
}

// TestPageList_CrossGroupGlobalPaging 校验不指定 group 时的全局分页：
// 2 组共 25 条、pageSize=10 时三页应为 10/10/5，total=25，跨页无重复无遗漏。
func TestPageList_CrossGroupGlobalPaging(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	// notify 造 15 条，security.login 造 10 条，共 25 条。
	for i := 0; i < 15; i++ {
		require.NoError(t, svc.Create(ctx, &dtosetting.SettingCreateReq{
			Group: "notify", Key: fmt.Sprintf("notify.k%02d", i), Value: "v", ValueType: "string",
		}))
	}
	for i := 0; i < 10; i++ {
		require.NoError(t, svc.Create(ctx, &dtosetting.SettingCreateReq{
			Group: "security.login", Key: fmt.Sprintf("sec.k%02d", i), Value: "1", ValueType: "int",
		}))
	}

	seen := map[string]bool{}
	var collected int

	page1, err := svc.PageList(ctx, &dtosetting.SettingPageListReq{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(25), page1.Total)
	assert.Len(t, page1.List, 10)

	page2, err := svc.PageList(ctx, &dtosetting.SettingPageListReq{Page: 2, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(25), page2.Total)
	assert.Len(t, page2.List, 10)

	page3, err := svc.PageList(ctx, &dtosetting.SettingPageListReq{Page: 3, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(25), page3.Total)
	assert.Len(t, page3.List, 5)

	for _, p := range []*dtosetting.SettingPageListResp{page1, page2, page3} {
		for _, it := range p.List {
			uid := it.Group + ":" + it.Key
			assert.Falsef(t, seen[uid], "跨页出现重复项 %s", uid)
			seen[uid] = true
			collected++
		}
	}
	assert.Equal(t, 25, collected, "三页合计应覆盖全部 25 条")
}

// TestPageList_SingleGroupPaging 校验指定 group 时单组原生分页。
func TestPageList_SingleGroupPaging(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		require.NoError(t, svc.Create(ctx, &dtosetting.SettingCreateReq{
			Group: "notify", Key: fmt.Sprintf("k%02d", i), Value: "v", ValueType: "string",
		}))
	}
	page, err := svc.PageList(ctx, &dtosetting.SettingPageListReq{Group: "notify", Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), page.Total)
	assert.Len(t, page.List, 2)
}
