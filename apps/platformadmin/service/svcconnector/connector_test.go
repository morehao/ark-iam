package svcconnector

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoconnector"
	"github.com/morehao/ark-iam/apps/platformadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func setupConnectorDB(t *testing.T) context.Context {
	t.Helper()
	testutil.SetupSQLite(t, &model.ConnectorEntity{}, &model.TenantEntity{})
	// 平台侧跨租户运维：连接器全局无归属租户，作用域显式声明为「全租户」。
	return dbclient.CrossTenantContext(context.Background())
}

func seedTenant(t *testing.T, ctx context.Context, tenantID, name string) {
	t.Helper()
	db := dbclient.IamDB(ctx)
	if err := db.WithContext(ctx).Create(&model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: tenantID}},
		Code:       "code-" + tenantID,
		Name:       name,
		Status:     model.TenantStatusActive,
	}).Error; err != nil {
		t.Fatalf("seed tenant %s: %v", tenantID, err)
	}
}

// createReq 构造一个合法（github oauth2）创建请求。
func createReq(tenantID string) *dtoconnector.ConnectorCreateReq {
	return &dtoconnector.ConnectorCreateReq{
		DefaultTenantID: tenantID,
		Name:            "connector-1",
		DisplayName:     "Connector 1",
		Protocol:        model.ConnectorProtocolOAuth2,
		Provider:        model.ConnectorProviderGithub,
		Status:          model.ConnectorStatusEnable,
		Config: model.ConnectorConfig{
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			ClientID:     "client-id",
			ClientSecret: "super-secret",
			RedirectURI:  "https://iam.example.com/callback",
		},
	}
}

// TestCreateAndDetail 跨租户创建 + 详情：default_tenant_id 与 defaultTenantName 回填，secret 脱敏。
func TestCreateAndDetail(t *testing.T) {
	ctx := setupConnectorDB(t)
	seedTenant(t, ctx, "t1", "Tenant One")
	svc := NewConnectorSvc(nil)

	resp, err := svc.Create(ctx, createReq("t1"))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if resp.ConnectorID == "" {
		t.Fatal("expected connector id")
	}

	detail, err := svc.Detail(ctx, &dtoconnector.ConnectorDetailReq{ConnectorID: resp.ConnectorID})
	if err != nil {
		t.Fatalf("Detail returned error: %v", err)
	}
	if detail.DefaultTenantID != "t1" {
		t.Fatalf("expected defaultTenantID t1, got %q", detail.DefaultTenantID)
	}
	if detail.DefaultTenantName != "Tenant One" {
		t.Fatalf("expected defaultTenantName 'Tenant One', got %q", detail.DefaultTenantName)
	}
	// secret 脱敏：详情不回传明文
	if detail.Config.ClientSecret != "******" {
		t.Fatalf("expected masked client secret, got %q", detail.Config.ClientSecret)
	}
	if detail.CreatedAt <= 0 || detail.UpdatedAt <= 0 {
		t.Fatalf("expected created/updated timestamps > 0, got %d/%d", detail.CreatedAt, detail.UpdatedAt)
	}
}

// TestCreateRequiresDefaultTenantWhenAutoCreate 开启自动开户但未指定默认落租户 -> ConnectorDefaultTenantRequiredError。
func TestCreateRequiresDefaultTenantWhenAutoCreate(t *testing.T) {
	ctx := setupConnectorDB(t)
	svc := NewConnectorSvc(nil)

	req := createReq("")
	req.AllowAutoCreateUser = model.ConnectorAutoCreateUserFlagEnable
	_, err := svc.Create(ctx, req)
	if err != code.GetError(code.ConnectorDefaultTenantRequiredError) {
		t.Fatalf("expected ConnectorDefaultTenantRequiredError, got %v", err)
	}
}

// TestCreateRequiresExistingDefaultTenant auto-create 指定了不存在的租户 -> ConnectorDefaultTenantRequiredError。
func TestCreateRequiresExistingDefaultTenant(t *testing.T) {
	ctx := setupConnectorDB(t)
	svc := NewConnectorSvc(nil)

	req := createReq("no-such-tenant")
	req.AllowAutoCreateUser = model.ConnectorAutoCreateUserFlagEnable
	_, err := svc.Create(ctx, req)
	if err != code.GetError(code.ConnectorDefaultTenantRequiredError) {
		t.Fatalf("expected ConnectorDefaultTenantRequiredError, got %v", err)
	}
}

// TestCreateRejectsIllegalStatus 非法状态被拒。
func TestCreateRejectsIllegalStatus(t *testing.T) {
	ctx := setupConnectorDB(t)
	svc := NewConnectorSvc(nil)

	req := createReq("")
	req.Status = "enabled"
	_, err := svc.Create(ctx, req)
	if err != code.GetError(code.ConnectorCreateError) {
		t.Fatalf("expected ConnectorCreateError, got %v", err)
	}
}

// TestPageListCrossTenantFilters 跨租户列表 + 名称回填 + 按 defaultTenantID 过滤。
func TestPageListCrossTenantFilters(t *testing.T) {
	ctx := setupConnectorDB(t)
	seedTenant(t, ctx, "t1", "Tenant One")
	seedTenant(t, ctx, "t2", "Tenant Two")
	svc := NewConnectorSvc(nil)

	if _, err := svc.Create(ctx, createReq("t1")); err != nil {
		t.Fatalf("create t1: %v", err)
	}
	req2 := createReq("t2")
	req2.Name = "connector-2"
	if _, err := svc.Create(ctx, req2); err != nil {
		t.Fatalf("create t2: %v", err)
	}

	all, err := svc.PageList(ctx, &dtoconnector.ConnectorPageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("PageList all: %v", err)
	}
	if all.Total != 2 {
		t.Fatalf("expected total 2, got %d", all.Total)
	}
	for _, item := range all.List {
		if item.DefaultTenantName == "" {
			t.Fatalf("expected defaultTenantName populated, got empty for %s", item.ConnectorID)
		}
	}

	filtered, err := svc.PageList(ctx, &dtoconnector.ConnectorPageListReq{Page: 1, PageSize: 10, DefaultTenantID: "t2"})
	if err != nil {
		t.Fatalf("PageList filtered: %v", err)
	}
	if filtered.Total != 1 || filtered.List[0].DefaultTenantID != "t2" {
		t.Fatalf("expected single t2 connector, got %+v", filtered.List)
	}
}

// TestUpdatePreservesSecret Update 时前端回传掩码 secret，应保留库内真实 secret。
func TestUpdatePreservesSecret(t *testing.T) {
	ctx := setupConnectorDB(t)
	seedTenant(t, ctx, "t1", "Tenant One")
	svc := NewConnectorSvc(nil)

	resp, err := svc.Create(ctx, createReq("t1"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	detail, err := svc.Detail(ctx, &dtoconnector.ConnectorDetailReq{ConnectorID: resp.ConnectorID})
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	// 模拟前端拿详情（masked）后再提交更新
	updateReq := &dtoconnector.ConnectorUpdateReq{
		ConnectorID:     resp.ConnectorID,
		DefaultTenantID: "t1",
		Name:            "connector-renamed",
		DisplayName:     detail.DisplayName,
		Protocol:        detail.Protocol,
		Provider:        detail.Provider,
		Status:          model.ConnectorStatusEnable,
		Config: model.ConnectorConfig{
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			ClientID:     "client-id",
			ClientSecret: "******", // 前端回传掩码
			RedirectURI:  "https://iam.example.com/callback",
		},
	}
	if err := svc.Update(ctx, updateReq); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// 直接从库读原始实体，验证真实 secret 未被掩码覆盖（绕过脱敏读取）
	db := dbclient.IamDB(ctx)
	var entity model.ConnectorEntity
	if err := db.WithContext(ctx).Where("id = ?", resp.ConnectorID).First(&entity).Error; err != nil {
		t.Fatalf("reload entity: %v", err)
	}
	if entity.Config.ClientSecret != "super-secret" {
		t.Fatalf("expected secret preserved as 'super-secret', got %q", entity.Config.ClientSecret)
	}
	if entity.Name != "connector-renamed" {
		t.Fatalf("expected name updated, got %q", entity.Name)
	}
}

// TestDelete 删除 + 不存在报错。
func TestDelete(t *testing.T) {
	ctx := setupConnectorDB(t)
	svc := NewConnectorSvc(nil)

	resp, err := svc.Create(ctx, createReq(""))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, &dtoconnector.ConnectorDeleteReq{ConnectorID: resp.ConnectorID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Detail(ctx, &dtoconnector.ConnectorDetailReq{ConnectorID: resp.ConnectorID}); err != code.GetError(code.ConnectorNotExistError) {
		t.Fatalf("expected ConnectorNotExistError after delete, got %v", err)
	}
	if err := svc.Delete(ctx, &dtoconnector.ConnectorDeleteReq{ConnectorID: "missing"}); err != code.GetError(code.ConnectorNotExistError) {
		t.Fatalf("expected ConnectorNotExistError for missing, got %v", err)
	}
}

// TestListFactories 工厂列表 + protocol/provider 过滤。
func TestListFactories(t *testing.T) {
	ctx := setupConnectorDB(t)
	svc := NewConnectorSvc(nil)

	resp, err := svc.ListFactories(ctx, &dtoconnector.ConnectorFactoryListReq{})
	if err != nil {
		t.Fatalf("ListFactories: %v", err)
	}
	if len(resp.List) < 3 {
		t.Fatalf("expected at least 3 factories, got %d", len(resp.List))
	}
	filtered, err := svc.ListFactories(ctx, &dtoconnector.ConnectorFactoryListReq{Protocol: model.ConnectorProtocolOIDC, Provider: model.ConnectorProviderGoogle})
	if err != nil {
		t.Fatalf("ListFactories filtered: %v", err)
	}
	if len(filtered.List) != 1 || filtered.List[0].Provider != model.ConnectorProviderGoogle {
		t.Fatalf("expected single google factory, got %+v", filtered.List)
	}
}

// TestTestConnector 连通性测试走驱动 TestConnection（stub OIDC/oauth2 均返回成功）。
func TestTestConnector(t *testing.T) {
	ctx := setupConnectorDB(t)
	svc := NewConnectorSvc(nil)

	resp, err := svc.Create(ctx, createReq(""))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	res, err := svc.TestConnector(ctx, &dtoconnector.ConnectorIDReq{ConnectorID: resp.ConnectorID})
	if err != nil {
		t.Fatalf("TestConnector: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success test result, got %+v", res)
	}
	if _, err := svc.TestConnector(ctx, &dtoconnector.ConnectorIDReq{ConnectorID: "missing"}); err != code.GetError(code.ConnectorNotExistError) {
		t.Fatalf("expected ConnectorNotExistError, got %v", err)
	}
}
