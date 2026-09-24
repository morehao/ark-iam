package svcapplicationclient

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

// TestUpdateBuiltInClientAllowsName 内置 OAuth 客户端的名称归运维（字段权威矩阵 create_only）：
// 控制台可改且重启不被种子回写；回调地址等运行参数一直归运维。
//
// 回归背景：名称一度是 reconcile，控制台改名直接报"不可修改"；reconcile 收窄到
// 「定位键 + 安全不变式」后（客户端只剩 source/app_id），名称交还运维。
func TestUpdateBuiltInClientAllowsName(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	builtin := &model.ApplicationClientEntity{
		TenantID: "1", AppID: "app1", Code: model.SeedBuiltinClientPlatformAdminWeb, Name: "平台管理后台",
		Source:                  model.ApplicationClientSourceBuiltin,
		Status:                  model.ApplicationClientStatusEnable,
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodNone,
		RequirePKCE:             model.ClientPKCEPolicyEnable,
	}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewApplicationClientSvc()

	// 改名 + 改回调地址：一并放行
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: builtin.ID,
		Name:                "运维自定名",
		Status:              model.ApplicationClientStatusEnable,
		RedirectURIs:        []string{"https://sso.example.com/auth/callback"},
	}); err != nil {
		t.Fatalf("内置客户端展示字段应可改: %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "运维自定名" {
		t.Fatalf("改名未落库, got %q", got.Name)
	}
	// JSON 列经 serializer 落库后读回即为具名切片，直接断言元素本身
	if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "https://sso.example.com/auth/callback" {
		t.Fatalf("回调地址未落库: %v", got.RedirectURIs)
	}
	// source（安全不变式）不在 Update 请求里，控制台无法改写内置标记
	if got.Source != model.ApplicationClientSourceBuiltin {
		t.Fatalf("source 不得被控制台改写: %q", got.Source)
	}
	// 请求未提交认证方式 / PKCE 策略时**必须保持存量**，不得被写成空串或降级为 disable。
	// 回归背景（原缺陷）：这两列曾在 fields 白名单里无条件全量写入，任何省略它们的 PUT 都会把
	// token_endpoint_auth_method 刷成空串——空串在 oidcop 侧落到 fail-closed 的 private_key_jwt
	// 分支，该控制台登录直接不可用；require_pkce 则被静默降级、丢掉公共客户端唯一的补偿控制。
	if got.TokenEndpointAuthMethod != model.TokenEndpointAuthMethodNone {
		t.Fatalf("内置客户端认证方式不得被改写: %q", got.TokenEndpointAuthMethod)
	}
	if got.RequirePKCE != model.ClientPKCEPolicyEnable {
		t.Fatalf("内置客户端强制 PKCE 不得被降级: %q", got.RequirePKCE)
	}
}

// TestUpdateClientKeepsJSONColumnsValidWhenOmitted 更新请求未提交的 JSON 列（nil）必须落成空数组。
// 回归背景：GORM 对 NOT NULL 的 JSON 列会把 nil 切片序列化为空串，而更新路径用 Select 显式列出列，
// nil 字段会被写进去——空串在 PostgreSQL 的 json 列上是非法 JSON，客户端详情还会把 [] 读成 null。
func TestUpdateClientKeepsJSONColumnsValidWhenOmitted(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	ctx := newOAuthDeleteCtx("1", "0")
	client := newTestClientEntity("客户端1", "client_1", model.ApplicationClientSourceThirdParty)
	client.AppID = "app1"
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := NewApplicationClientSvc().Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: client.ID,
		Name:                "客户端1改名",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := dao.NewApplicationClientDao().GetByID(ctx, client.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RedirectURIs == nil || got.GrantTypes == nil || got.ResponseTypes == nil {
		t.Fatalf("未提交的 JSON 列应为空切片而非 nil: %+v", got)
	}

	// 原始列值必须是合法 JSON（NOT NULL 的 json 列不得出现空串）
	var raw struct {
		RedirectURIs string `gorm:"column:redirect_uris"`
	}
	if err := db.WithContext(ctx).Raw("select redirect_uris from application_client where id = ?", client.ID).Scan(&raw).Error; err != nil {
		t.Fatalf("raw query: %v", err)
	}
	if raw.RedirectURIs != "[]" {
		t.Fatalf("redirect_uris 原始列值 = %q, want []", raw.RedirectURIs)
	}
}
