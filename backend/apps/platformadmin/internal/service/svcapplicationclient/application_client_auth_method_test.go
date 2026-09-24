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

// 本文件覆盖「客户端类型不变式」的两条收口：
//  1. 内置（浏览器公共）客户端的 token_endpoint_auth_method / require_pkce 控制台拒改；
//  2. 公共客户端（none）不得签发客户端密钥。
//
// 规范依据：RFC 6749 §10.1（不得为 user-agent 类客户端签发/要求客户端凭据）、
// RFC 10017 §6.3.3.1（浏览器客户端 MUST 登记为 public，授权服务器 MUST NOT 对其要求客户端认证）。

// TestUpdateBuiltInClientRejectsAuthMethodChange 内置客户端不得被改成机密客户端：
// 前端只传 client_id、不持密钥，改成 basic/post 后 token 端点要求认证而前端无法提供，
// 该控制台登录与静默续期会全部 invalid_client，且修复入口就在被锁死的控制台内部。
func TestUpdateBuiltInClientRejectsAuthMethodChange(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	builtin := newTestClientEntity("平台管理后台", model.SeedBuiltinClientPlatformAdminWeb, model.ApplicationClientSourceBuiltin)
	builtin.AppID = "app1"
	builtin.TokenEndpointAuthMethod = model.TokenEndpointAuthMethodNone
	builtin.RequirePKCE = model.ClientPKCEPolicyEnable
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	err := NewApplicationClientSvc().Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID:     builtin.ID,
		Name:                    "平台管理后台",
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodBasic,
		RequirePKCE:             model.ClientPKCEPolicyEnable,
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientBuiltInAuthMethodImmutableError) {
		t.Fatalf("内置客户端改认证方式应返回 ApplicationClientBuiltInAuthMethodImmutableError, got %v", err)
	}

	got, err := dao.NewApplicationClientDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.TokenEndpointAuthMethod != model.TokenEndpointAuthMethodNone {
		t.Fatalf("认证方式不得被改写: %q", got.TokenEndpointAuthMethod)
	}
}

// TestUpdateBuiltInClientRejectsPKCEDowngrade 内置客户端的强制 PKCE 不得被关掉：
// 公共客户端没有密钥，PKCE 是唯一的授权码绑定手段，关掉等于放弃防降级。
func TestUpdateBuiltInClientRejectsPKCEDowngrade(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	builtin := newTestClientEntity("平台管理后台", model.SeedBuiltinClientPlatformAdminWeb, model.ApplicationClientSourceBuiltin)
	builtin.AppID = "app1"
	builtin.TokenEndpointAuthMethod = model.TokenEndpointAuthMethodNone
	builtin.RequirePKCE = model.ClientPKCEPolicyEnable
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	err := NewApplicationClientSvc().Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID:     builtin.ID,
		Name:                    "平台管理后台",
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodNone,
		RequirePKCE:             model.ClientPKCEPolicyDisable,
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientBuiltInAuthMethodImmutableError) {
		t.Fatalf("内置客户端关掉强制 PKCE 应返回 ApplicationClientBuiltInAuthMethodImmutableError, got %v", err)
	}

	got, err := dao.NewApplicationClientDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RequirePKCE != model.ClientPKCEPolicyEnable {
		t.Fatalf("强制 PKCE 不得被降级: %q", got.RequirePKCE)
	}
}

// TestUpdateThirdPartyClientCanSwitchAuthMethod 自建客户端不在内置不变式范围内：
// 从公共客户端迁到机密客户端（如改为 BFF）必须允许。
func TestUpdateThirdPartyClientCanSwitchAuthMethod(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	thirdParty := newTestClientEntity("第三方 SPA", "third_party_web", model.ApplicationClientSourceThirdParty)
	thirdParty.AppID = "app1"
	thirdParty.TokenEndpointAuthMethod = model.TokenEndpointAuthMethodNone
	thirdParty.RequirePKCE = model.ClientPKCEPolicyEnable
	if err := db.Create(thirdParty).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	if err := NewApplicationClientSvc().Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID:     thirdParty.ID,
		Name:                    "第三方 BFF",
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodBasic,
		RequirePKCE:             model.ClientPKCEPolicyDisable,
	}); err != nil {
		t.Fatalf("自建客户端应可切换认证方式: %v", err)
	}

	got, err := dao.NewApplicationClientDao().GetByID(ctx, thirdParty.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.TokenEndpointAuthMethod != model.TokenEndpointAuthMethodBasic || got.RequirePKCE != model.ClientPKCEPolicyDisable {
		t.Fatalf("切换未生效: method=%q pkce=%q", got.TokenEndpointAuthMethod, got.RequirePKCE)
	}
}

// TestUpdateAuthMethodEmptyKeepsExisting 认证方式留空表示「本次不修改」。
//
// 回归背景（原缺陷）：该列曾在 fields 白名单里无条件全量写入，任何省略它的 PUT 都会把存量
// 刷成空串——空串不是合法枚举，在 oidcop 侧落到 fail-closed 的 private_key_jwt 分支
// （client.go 的 default 分支），客户端随后无法用任何方式在令牌端点认证。
func TestUpdateAuthMethodEmptyKeepsExisting(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	thirdParty := newTestClientEntity("服务端应用", "server_side_app", model.ApplicationClientSourceThirdParty)
	thirdParty.AppID = "app1"
	thirdParty.TokenEndpointAuthMethod = model.TokenEndpointAuthMethodBasic
	if err := db.Create(thirdParty).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	// 只改名字，完全不提交认证方式
	if err := NewApplicationClientSvc().Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: thirdParty.ID,
		Name:                "服务端应用改名",
	}); err != nil {
		t.Fatalf("改名应允许: %v", err)
	}

	got, err := dao.NewApplicationClientDao().GetByID(ctx, thirdParty.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.TokenEndpointAuthMethod != model.TokenEndpointAuthMethodBasic {
		t.Fatalf("留空必须保持存量认证方式, got %q（被写成空串会让客户端静默不可用）", got.TokenEndpointAuthMethod)
	}
}

// TestUpdateRejectsIllegalAuthMethod 认证方式只允许白名单三值；未知取值会落到
// private_key_jwt 分支，必须在 service 入口 fail-closed。
func TestUpdateRejectsIllegalAuthMethod(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	thirdParty := newTestClientEntity("服务端应用", "server_side_app", model.ApplicationClientSourceThirdParty)
	thirdParty.AppID = "app1"
	if err := db.Create(thirdParty).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	err := NewApplicationClientSvc().Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID:     thirdParty.ID,
		Name:                    "服务端应用",
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethod("private_key_jwt"),
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientAuthMethodInvalidError) {
		t.Fatalf("非法认证方式应返回 ApplicationClientAuthMethodInvalidError, got %v", err)
	}
}

// TestCreateRejectsIllegalAuthMethod 创建路径同样按白名单校验（留空取列默认）。
func TestCreateRejectsIllegalAuthMethod(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationClientEntity{})

	ctx := newOAuthDeleteCtx("1", "0")
	_, err := NewApplicationClientSvc().Create(ctx, &dtoapplicationclient.ApplicationClientCreateReq{
		AppID: "app1", Code: "client_illegal_method", Name: "非法认证方式客户端",
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethod("client_secret_jwt"),
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientAuthMethodInvalidError) {
		t.Fatalf("创建非法认证方式应返回 ApplicationClientAuthMethodInvalidError, got %v", err)
	}
}

// TestCreateSecretRejectedForPublicClient 公共客户端（none）不得签发客户端密钥：
// 这类客户端的代码会下发给每个用户，密钥无处安全保存（RFC 6749 §2.1），
// 且令牌端点本就不会向其索取认证——签名等于制造一把用不上的凭据。
func TestCreateSecretRejectedForPublicClient(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{})
	public := newTestClientEntity("第三方 SPA", "public_spa", model.ApplicationClientSourceThirdParty)
	public.AppID = "app1"
	public.TokenEndpointAuthMethod = model.TokenEndpointAuthMethodNone
	public.RequirePKCE = model.ClientPKCEPolicyEnable
	if err := db.Create(public).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	_, err := NewApplicationClientSvc().CreateSecret(ctx, &dtoapplicationclient.SecretCreateReq{
		ApplicationClientID: public.ID, Name: "prod-key",
	})
	if err == nil || gerror.GetCode(err) != int(code.ApplicationClientPublicClientSecretForbiddenError) {
		t.Fatalf("公共客户端建密钥应返回 ApplicationClientPublicClientSecretForbiddenError, got %v", err)
	}

	secrets, err := dao.NewApplicationClientSecretDao().GetListByCond(ctx, &dao.ApplicationClientSecretCond{ApplicationClientID: public.ID})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(secrets) != 0 {
		t.Fatalf("被拒绝的请求不得落库，实际落库 %d 条", len(secrets))
	}
}

// TestCreateSecretAllowedForConfidentialClient 机密客户端仍可正常签发密钥（收口不得误伤）。
func TestCreateSecretAllowedForConfidentialClient(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{})
	confidential := newTestClientEntity("资源服务器", "resource_server", model.ApplicationClientSourceThirdParty)
	confidential.AppID = "app1"
	confidential.TokenEndpointAuthMethod = model.TokenEndpointAuthMethodBasic
	if err := db.Create(confidential).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "0")
	resp, err := NewApplicationClientSvc().CreateSecret(ctx, &dtoapplicationclient.SecretCreateReq{
		ApplicationClientID: confidential.ID, Name: "prod-key",
	})
	if err != nil {
		t.Fatalf("机密客户端建密钥应放行: %v", err)
	}
	if resp == nil || resp.Secret == "" || resp.ValuePrefix == "" {
		t.Fatalf("密钥明文与前缀应返回一次: %+v", resp)
	}
}
