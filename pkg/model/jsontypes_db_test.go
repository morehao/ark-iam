package model_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 本文件是 JSON 列（gorm:"serializer:json"）的落库形状 golden：
// 断言的是**数据库里的原始文本**，而不是读回来的 Go 值——只有原始文本能证明
// 写入走过 serializer（map 更新会静默落脏值，读回时才发现 invalid character）。

func openJSONGoldenDB(t *testing.T, entities ...any) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:json_golden_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(entities...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func rawColumn(t *testing.T, db *gorm.DB, table, column, id string) string {
	t.Helper()
	var got string
	if err := db.Raw(fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", column, table), id).Scan(&got).Error; err != nil {
		t.Fatalf("query raw %s.%s: %v", table, column, err)
	}
	return got
}

func TestApplicationClientJSONColumnsGolden(t *testing.T) {
	db := openJSONGoldenDB(t, &model.ApplicationClientEntity{})

	client := &model.ApplicationClientEntity{
		Code:                   "golden_client",
		Name:                   "golden",
		RedirectURIs:           model.RedirectURIList{"http://localhost:4001/auth/callback"},
		PostLogoutRedirectURIs: model.PostLogoutRedirectURIList{"http://localhost:4001/login"},
		GrantTypes:             model.GrantTypeList{model.GrantTypeAuthorizationCode, model.GrantTypeRefreshToken},
		ResponseTypes:          model.ResponseTypeList{model.ResponseTypeCode},
		AllowedOrigins:         model.AllowedOriginList{"https://console.example.com"},
		DefaultScopes:          model.DefaultScopeList{model.ScopeOpenID, model.ScopeProfile},
	}
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("create application_client: %v", err)
	}

	golden := map[string]string{
		"redirect_uris":             `["http://localhost:4001/auth/callback"]`,
		"post_logout_redirect_uris": `["http://localhost:4001/login"]`,
		"grant_types":               `["authorization_code","refresh_token"]`,
		"response_types":            `["code"]`,
		"allowed_origins":           `["https://console.example.com"]`,
		"default_scopes":            `["openid","profile"]`,
	}
	for column, want := range golden {
		if got := rawColumn(t, db, model.TableNameApplicationClient, column, client.ID); got != want {
			t.Errorf("%s raw text = %s, want %s", column, got, want)
		}
	}

	var loaded model.ApplicationClientEntity
	if err := db.First(&loaded, "id = ?", client.ID).Error; err != nil {
		t.Fatalf("reload application_client: %v", err)
	}
	if len(loaded.RedirectURIs) != 1 || loaded.RedirectURIs[0] != client.RedirectURIs[0] {
		t.Errorf("redirect uris round trip = %v", loaded.RedirectURIs)
	}
	if len(loaded.GrantTypes) != 2 || loaded.GrantTypes[1] != model.GrantTypeRefreshToken {
		t.Errorf("grant types round trip = %v", loaded.GrantTypes)
	}
	if len(loaded.DefaultScopes) != 2 || loaded.DefaultScopes[1] != model.ScopeProfile {
		t.Errorf("default scopes round trip = %v", loaded.DefaultScopes)
	}
}

// TestStructuredUpdateWritesValidJSON 结构化 Updates + Select 是 JSON 列唯一的更新路径：
// 原始文本必须仍是合法 JSON（map 更新不经过 serializer，会写出 Go map 字面量）。
func TestStructuredUpdateWritesValidJSON(t *testing.T) {
	db := openJSONGoldenDB(t, &model.ApplicationClientEntity{})

	client := &model.ApplicationClientEntity{Code: "update_client"}
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("create application_client: %v", err)
	}

	updated := &model.ApplicationClientEntity{
		RedirectURIs: model.RedirectURIList{"https://sso.example.com/auth/callback"},
		GrantTypes:   model.GrantTypeList{model.GrantTypeAuthorizationCode},
	}
	if err := db.Model(&model.ApplicationClientEntity{}).Where("id = ?", client.ID).
		Select("redirect_uris", "grant_types").Updates(updated).Error; err != nil {
		t.Fatalf("structured update: %v", err)
	}

	if got, want := rawColumn(t, db, model.TableNameApplicationClient, "redirect_uris", client.ID), `["https://sso.example.com/auth/callback"]`; got != want {
		t.Errorf("redirect_uris after update = %s, want %s", got, want)
	}
	if got, want := rawColumn(t, db, model.TableNameApplicationClient, "grant_types", client.ID), `["authorization_code"]`; got != want {
		t.Errorf("grant_types after update = %s, want %s", got, want)
	}

	var loaded model.ApplicationClientEntity
	if err := db.First(&loaded, "id = ?", client.ID).Error; err != nil {
		t.Fatalf("reload application_client: %v", err)
	}
	if len(loaded.RedirectURIs) != 1 || loaded.RedirectURIs[0] != "https://sso.example.com/auth/callback" {
		t.Errorf("redirect uris after update round trip = %v", loaded.RedirectURIs)
	}
}

func TestConnectorJSONColumnsGolden(t *testing.T) {
	db := openJSONGoldenDB(t, &model.ConnectorEntity{})

	connector := &model.ConnectorEntity{
		Name:     "golden connector",
		Protocol: model.ConnectorProtocolOIDC,
		Provider: model.ConnectorProviderMicrosoft,
		Config: model.ConnectorConfig{
			Protocol:     model.ConnectorProtocolOIDC,
			Provider:     model.ConnectorProviderMicrosoft,
			Issuer:       "https://login.microsoftonline.com/common/v2.0",
			ClientID:     "entra-client",
			ClientSecret: "secret",
			RedirectURI:  "https://iam.example.com/callback",
			Scopes:       model.ConnectorScopeList{model.ScopeOpenID, model.ScopeEmail},
			Tenant:       "common",
		},
		ClaimMapping: model.ConnectorClaimMapping{Email: "email", DisplayName: "name"},
		DomainPolicy: model.ConnectorDomainPolicy{AllowedDomains: model.DomainList{"example.com"}},
	}
	if err := db.Create(connector).Error; err != nil {
		t.Fatalf("create connector: %v", err)
	}

	config := rawColumn(t, db, model.TableNameConnector, "config", connector.ID)
	for _, fragment := range []string{`"protocol":"oidc"`, `"provider":"microsoft"`, `"clientID":"entra-client"`, `"tenant":"common"`, `"scopes":["openid","email"]`} {
		if !strings.Contains(config, fragment) {
			t.Errorf("config raw text %s missing %s", config, fragment)
		}
	}
	if got, want := rawColumn(t, db, model.TableNameConnector, "claim_mapping", connector.ID), `{"email":"email","displayName":"name"}`; got != want {
		t.Errorf("claim_mapping raw text = %s, want %s", got, want)
	}
	if got, want := rawColumn(t, db, model.TableNameConnector, "domain_policy", connector.ID), `{"allowedDomains":["example.com"]}`; got != want {
		t.Errorf("domain_policy raw text = %s, want %s", got, want)
	}

	var loaded model.ConnectorEntity
	if err := db.First(&loaded, "id = ?", connector.ID).Error; err != nil {
		t.Fatalf("reload connector: %v", err)
	}
	if loaded.Config.Tenant != "common" || loaded.Config.Scopes[1] != model.ScopeEmail {
		t.Errorf("connector config round trip = %+v", loaded.Config)
	}
	if loaded.DomainPolicy.AllowedDomains[0] != "example.com" {
		t.Errorf("connector domain policy round trip = %+v", loaded.DomainPolicy)
	}
}

// TestNullableJSONColumnKeepsNullShape：可空 JSON 列（refresh_token.scopes/amr）不传值时为 NULL，
// 而不是空串——空串在 PostgreSQL 的 json 列上非法。写入侧必须保证非空值否则显式落空数组。
func TestNullableJSONColumnKeepsNullShape(t *testing.T) {
	db := openJSONGoldenDB(t, &model.RefreshTokenEntity{})

	entity := &model.RefreshTokenEntity{Token: "hash"}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("create refresh_token: %v", err)
	}

	var nullScopes, nullAMR *string
	if err := db.Raw(fmt.Sprintf("SELECT scopes, amr FROM %s WHERE id = ?", model.TableNameRefreshToken), entity.ID).
		Row().Scan(&nullScopes, &nullAMR); err != nil {
		t.Fatalf("scan raw nullable json: %v", err)
	}
	if nullScopes != nil || nullAMR != nil {
		t.Fatalf("nil 切片应落 NULL，实际 scopes=%v amr=%v", nullScopes, nullAMR)
	}

	var loaded model.RefreshTokenEntity
	if err := db.First(&loaded, "id = ?", entity.ID).Error; err != nil {
		t.Fatalf("reload refresh_token: %v", err)
	}
	if loaded.Scopes != nil || loaded.AMR != nil {
		t.Fatalf("NULL 应回读为 nil，实际 %#v/%#v", loaded.Scopes, loaded.AMR)
	}
}

func TestRefreshTokenAndIdentityJSONColumnsGolden(t *testing.T) {
	db := openJSONGoldenDB(t, &model.RefreshTokenEntity{}, &model.UserIdentityEntity{}, &model.ApplicationEntity{})

	refreshToken := &model.RefreshTokenEntity{
		Token:  "hash",
		Scopes: model.ScopeList{model.ScopeOpenID, model.ScopeProfile},
		AMR:    model.AuthMethodList{"pwd"},
	}
	if err := db.Create(refreshToken).Error; err != nil {
		t.Fatalf("create refresh_token: %v", err)
	}
	if got, want := rawColumn(t, db, model.TableNameRefreshToken, "scopes", refreshToken.ID), `["openid","profile"]`; got != want {
		t.Errorf("scopes raw text = %s, want %s", got, want)
	}
	if got, want := rawColumn(t, db, model.TableNameRefreshToken, "amr", refreshToken.ID), `["pwd"]`; got != want {
		t.Errorf("amr raw text = %s, want %s", got, want)
	}

	identity := &model.UserIdentityEntity{
		PersonID:        "person-1",
		Issuer:          "https://issuer.example.com",
		ExternalSubject: "sub-1",
		Detail: model.UserIdentityDetail{
			Issuer:        "https://issuer.example.com",
			Subject:       "sub-1",
			Email:         "user@example.com",
			EmailVerified: model.EmailVerificationVerified,
		},
	}
	if err := db.Create(identity).Error; err != nil {
		t.Fatalf("create user_identity: %v", err)
	}
	if got, want := rawColumn(t, db, model.TableNameUserIdentity, "detail", identity.ID),
		`{"issuer":"https://issuer.example.com","subject":"sub-1","email":"user@example.com","emailVerified":"verified"}`; got != want {
		t.Errorf("detail raw text = %s, want %s", got, want)
	}

	app := &model.ApplicationEntity{
		Code:         "golden_app",
		Name:         "golden app",
		RoleTemplate: model.RoleTemplateItemList{{Code: "storage_admin", Name: "存储管理员"}},
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}
	if got, want := rawColumn(t, db, model.TableNameApplication, "role_template", app.ID),
		`[{"code":"storage_admin","name":"存储管理员"}]`; got != want {
		t.Errorf("role_template raw text = %s, want %s", got, want)
	}

	var loadedApp model.ApplicationEntity
	if err := db.First(&loadedApp, "id = ?", app.ID).Error; err != nil {
		t.Fatalf("reload application: %v", err)
	}
	if len(loadedApp.RoleTemplate) != 1 || loadedApp.RoleTemplate[0].Code != "storage_admin" {
		t.Errorf("role_template round trip = %+v", loadedApp.RoleTemplate)
	}
}
