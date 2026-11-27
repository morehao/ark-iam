package svcapplicationclient

import (
	"testing"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/apps/platformadmin/testutil"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
)

// seedOwnedApp 造一条应用（全局表，无 tenant_id，不需要作用域）。
func seedOwnedApp(t *testing.T, ownerType model.AppOwnerType, ownerTenantID string) *model.ApplicationEntity {
	t.Helper()
	app := &model.ApplicationEntity{
		Code:          "app_" + ownerTenantID + string(ownerType),
		Name:          "归属测试应用",
		OwnerType:     ownerType,
		OwnerTenantID: ownerTenantID,
		RoleTemplate:  model.RoleTemplateItemList{},
	}
	if err := dao.NewApplicationDao().Insert(dbclient.CrossTenantContext(t.Context()), app); err != nil {
		t.Fatalf("seed app: %v", err)
	}
	return app
}

func clientTenantID(t *testing.T, clientID string) string {
	t.Helper()
	got, err := dao.NewApplicationClientDao().GetByID(dbclient.CrossTenantContext(t.Context()), clientID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatalf("客户端未落库: %s", clientID)
	}
	return got.TenantID
}

// TestCreateTenantOwnedClientLandsOwnerTenant 平台代租户建 client：请求来自平台作用域（平台/AllScope），
// 但应用 owner_type=tenant，client.tenant_id 必须落到 app.owner_tenant_id，而不是平台/空串。
// 这是评审 H5 的核心回归：不改则租户侧永远查不到。
func TestCreateTenantOwnedClientLandsOwnerTenant(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})

	app := seedOwnedApp(t, model.AppOwnerTypeTenant, "tenant_a")

	// 平台代建：作用域是平台（AllScope 时 TenantIDFromContext 返回空串）
	platformCtx := newOAuthDeleteCtx("", "0")
	resp, err := NewApplicationClientSvc(nil).Create(requestCtx(platformCtx), &dtoapplicationclient.ApplicationClientCreateReq{
		AppID: app.ID, Code: "tenant_a_client", Name: "租户A接入客户端",
		Operator: operatorOf(platformCtx),
	})
	if err != nil {
		t.Fatalf("创建租户归属 client 应成功: %v", err)
	}

	got := clientTenantID(t, resp.ApplicationClientID)
	if got != app.OwnerTenantID {
		t.Fatalf("client.tenant_id = %q, want owner_tenant_id %q（平台代建必须落租户作用域）", got, app.OwnerTenantID)
	}

	// 租户侧用本租户作用域应查得到
	tenantCtx := dbclient.CurrentTenantContext(t.Context(), "tenant_a")
	visible, err := dao.NewApplicationClientDao().GetByID(tenantCtx, resp.ApplicationClientID)
	if err != nil {
		t.Fatalf("租户作用域 GetByID: %v", err)
	}
	if visible == nil || visible.ID == "" {
		t.Fatalf("租户侧应查得到归属本租户的 client")
	}

	// 其它租户作用域不可见（fail-closed 过滤生效）
	otherCtx := dbclient.CurrentTenantContext(t.Context(), "tenant_b")
	hidden, err := dao.NewApplicationClientDao().GetByID(otherCtx, resp.ApplicationClientID)
	if err != nil {
		t.Fatalf("其它租户作用域 GetByID: %v", err)
	}
	if hidden != nil && hidden.ID != "" {
		t.Fatalf("其它租户不得看到该 client, got %+v", hidden)
	}
}

// TestCreatePlatformOwnedClientKeepsCurrentBehavior 平台应用（owner_type=platform）保持现行为：
// tenant_id 仍取当前请求作用域（TenantIDFromContext），不被 owner 推导覆盖。
func TestCreatePlatformOwnedClientKeepsCurrentBehavior(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})

	app := seedOwnedApp(t, model.AppOwnerTypePlatform, "")

	ctx := newOAuthDeleteCtx("tenant_x", "0")
	resp, err := NewApplicationClientSvc(nil).Create(requestCtx(ctx), &dtoapplicationclient.ApplicationClientCreateReq{
		AppID: app.ID, Code: "platform_client", Name: "平台应用客户端",
		Operator: operatorOf(ctx),
	})
	if err != nil {
		t.Fatalf("创建平台应用 client 应成功: %v", err)
	}

	if got := clientTenantID(t, resp.ApplicationClientID); got != "tenant_x" {
		t.Fatalf("client.tenant_id = %q, want tenant_x（平台应用保持取当前作用域）", got)
	}
}

// TestCreateClientWithMissingAppKeepsCurrentBehavior 找不到应用时保持现行为（tenant_id 取当前作用域），
// 不做归属强校验（不因取不到应用而失败）。
func TestCreateClientWithMissingAppKeepsCurrentBehavior(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})

	ctx := newOAuthDeleteCtx("tenant_y", "0")
	resp, err := NewApplicationClientSvc(nil).Create(requestCtx(ctx), &dtoapplicationclient.ApplicationClientCreateReq{
		AppID: "no_such_app", Code: "orphan_client", Name: "无应用客户端",
		Operator: operatorOf(ctx),
	})
	if err != nil {
		t.Fatalf("应用不存在应保持现行为放行: %v", err)
	}
	if got := clientTenantID(t, resp.ApplicationClientID); got != "tenant_y" {
		t.Fatalf("client.tenant_id = %q, want tenant_y（无应用时保持取当前作用域）", got)
	}
}

// TestCreateBuiltinClientUnaffected 内置客户端不受本改动影响：种子写入的 client 归属与认证方式保持，
// 且内置客户端不经 Create 控制台路径（其 tenant_id 属于平台侧既定配置，不被 owner 推导覆盖）。
func TestCreateBuiltinClientUnaffected(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})

	platformApp := seedOwnedApp(t, model.AppOwnerTypeTenant, "t_platform")

	// 种子写入的内置客户端（旧口径 tenant_id=平台租户）
	builtin := &model.ApplicationClientEntity{
		TenantID: "t_platform", AppID: platformApp.ID, Code: model.SeedBuiltinClientPlatformAdminWeb,
		Name: "平台管理后台", Source: model.ApplicationClientSourceBuiltin,
		Status:                  model.ApplicationClientStatusEnable,
		TokenEndpointAuthMethod: model.TokenEndpointAuthMethodNone,
		RequirePKCE:             model.ClientPKCEPolicyEnable,
	}
	if err := dao.NewApplicationClientDao().Insert(dbclient.CrossTenantContext(t.Context()), builtin); err != nil {
		t.Fatalf("seed builtin client: %v", err)
	}

	// 内置客户端可被其归属平台租户看到，认证方式/PKCE 不变
	got, err := dao.NewApplicationClientDao().GetByID(dbclient.CurrentTenantContext(t.Context(), "t_platform"), builtin.ID)
	if err != nil || got == nil || got.ID == "" {
		t.Fatalf("内置客户端应可见: %v, %+v", err, got)
	}
	if got.TenantID != "t_platform" {
		t.Fatalf("内置客户端 tenant_id 不应被改写: %q", got.TenantID)
	}
	if got.TokenEndpointAuthMethod != model.TokenEndpointAuthMethodNone || got.RequirePKCE != model.ClientPKCEPolicyEnable {
		t.Fatalf("内置客户端安全不变式不应被改写: auth=%q pkce=%q", got.TokenEndpointAuthMethod, got.RequirePKCE)
	}
}
