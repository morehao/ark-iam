package svcapplicationclient

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gerror"
)

// TestCreateRejectsInvalidClientCode 客户端编码由创建方填写，service 入口按 model.ClientCodePattern
// 拦截非法值（连字符/数字/大写/下划线开头/空格/空）：编码即 OIDC client_id，同时是网关 audience
// 白名单值，写错一个字符会让该客户端签发的令牌全部 401，必须在落库前拦下。
// 与 svcapplication 的应用编码校验同款（校验归 service，返回领域错误码）。
func TestCreateRejectsInvalidClientCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	ctx := newOAuthDeleteCtx("1", "0")
	svc := NewApplicationClientSvc()

	invalidCodes := []string{
		"platform-admin-web", // 连字符
		"client1",            // 数字
		"Client_Web",         // 大写
		"_client",            // 下划线开头
		"client web",         // 空格
		"client.web",         // 点号
		"",
	}
	for _, badCode := range invalidCodes {
		_, err := svc.Create(ctx, &dtoapplicationclient.ApplicationClientCreateReq{
			AppID: "app1", Code: badCode, Name: "非法编码客户端",
		})
		if err == nil {
			t.Fatalf("Create(code=%q) 必须被拒绝", badCode)
		}
		if gerror.GetCode(err) != int(code.ApplicationClientCodeInvalidError) {
			t.Fatalf("Create(code=%q) error code = %d, want %d",
				badCode, gerror.GetCode(err), code.ApplicationClientCodeInvalidError)
		}
	}

	var count int64
	if err := db.Model(&model.ApplicationClientEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count application_client: %v", err)
	}
	if count != 0 {
		t.Fatalf("被拒的创建不得落库, count=%d", count)
	}
}

// TestCreateAcceptsClientCodeAsClientID 合法编码（小写字母 + 下划线）落库为 client_id，
// 且取自入参——控制台不再由服务端随机生成编码。
func TestCreateAcceptsClientCodeAsClientID(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	ctx := newOAuthDeleteCtx("1", "0")

	resp, err := NewApplicationClientSvc().Create(ctx, &dtoapplicationclient.ApplicationClientCreateReq{
		AppID: "app1", Code: "iam_client", Name: "接入客户端",
	})
	if err != nil {
		t.Fatalf("合法编码应放行: %v", err)
	}
	if resp.Code != "iam_client" {
		t.Fatalf("resp code = %q, want iam_client", resp.Code)
	}

	got, err := dao.NewApplicationClientDao().GetByID(ctx, resp.ApplicationClientID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatalf("客户端未落库: %+v", got)
	}
	if got.Code != "iam_client" {
		t.Errorf("stored code = %q, want iam_client（编码取自入参，不再随机生成）", got.Code)
	}
	if got.Source != model.ApplicationClientSourceThirdParty {
		t.Errorf("stored source = %q, want %q", got.Source, model.ApplicationClientSourceThirdParty)
	}
}

// TestUpdateClientAllowsCodeRenameForThirdParty 用户自建客户端的编码可改（种子不管它们）：
// 改名后该 RP 需同步自己的 client_id 配置、旧令牌按新 aud 失效——这是 OIDC 固有语义。
func TestUpdateClientAllowsCodeRenameForThirdParty(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	ctx := newOAuthDeleteCtx("1", "0")
	svc := NewApplicationClientSvc()

	created, err := svc.Create(ctx, &dtoapplicationclient.ApplicationClientCreateReq{
		AppID: "app1", Code: "iam_client", Name: "接入客户端",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: created.ApplicationClientID,
		Code:                "iam_client_web",
		Name:                "接入客户端",
		Status:              model.ApplicationClientStatusEnable,
	}); err != nil {
		t.Fatalf("自建客户端改编码应放行: %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, created.ApplicationClientID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Code != "iam_client_web" {
		t.Fatalf("编码未落库: %q", got.Code)
	}

	// 改回原编码（回传未变化的编码同样放行）
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: created.ApplicationClientID,
		Code:                "iam_client_web",
		Name:                "接入客户端",
		Status:              model.ApplicationClientStatusEnable,
	}); err != nil {
		t.Fatalf("编码未变化应放行: %v", err)
	}

	// 非法编码被拒
	err = svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: created.ApplicationClientID,
		Code:                "iam-client-v3",
		Name:                "接入客户端",
		Status:              model.ApplicationClientStatusEnable,
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientCodeInvalidError) {
		t.Fatalf("非法编码必须被拒绝, got %v", err)
	}
}

// TestUpdateBuiltInClientRejectsCodeRename 内置控制台客户端的编码（= client_id）保持只读：
// 它同时是网关 aud 白名单与前端构建期默认值，从控制台改名会当场把该控制台锁死且界面无法自救。
func TestUpdateBuiltInClientRejectsCodeRename(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	ctx := newOAuthDeleteCtx("1", "0")
	svc := NewApplicationClientSvc()

	builtin := &model.ApplicationClientEntity{
		TenantID: "1", AppID: "app1", Code: model.SeedBuiltinClientPlatformAdminWeb, Name: "平台管理后台",
		Source: model.ApplicationClientSourceBuiltin, Status: model.ApplicationClientStatusEnable,
	}
	if err := dao.NewApplicationClientDao().Insert(ctx, builtin); err != nil {
		t.Fatalf("seed builtin client: %v", err)
	}

	err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: builtin.ID,
		Code:                "platform_console_web",
		Name:                builtin.Name,
		Status:              model.ApplicationClientStatusEnable,
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientBuiltInCodeImmutableError) {
		t.Fatalf("内置客户端改编码必须被拒绝, got %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Code != model.SeedBuiltinClientPlatformAdminWeb {
		t.Fatalf("被拒的改名不得落库: %q", got.Code)
	}

	// 回传未变化的编码不受影响（幂等友好），其它字段照常可改
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: builtin.ID,
		Code:                model.SeedBuiltinClientPlatformAdminWeb,
		Name:                "运维自定名",
		Status:              model.ApplicationClientStatusEnable,
	}); err != nil {
		t.Fatalf("编码未变化时其余字段应可改: %v", err)
	}
}
