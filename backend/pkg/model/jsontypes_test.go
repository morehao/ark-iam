package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestJSONCarrierSliceMarshalShape(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "redirect uris nil", value: RedirectURIList(nil), want: "null"},
		{name: "redirect uris empty", value: RedirectURIList{}, want: "[]"},
		{name: "redirect uris values", value: RedirectURIList{"https://a/cb"}, want: `["https://a/cb"]`},
		{name: "post logout nil", value: PostLogoutRedirectURIList(nil), want: "null"},
		{name: "post logout empty", value: PostLogoutRedirectURIList{}, want: "[]"},
		{name: "allowed origins nil", value: AllowedOriginList(nil), want: "null"},
		{name: "allowed origins empty", value: AllowedOriginList{}, want: "[]"},
		{name: "default scopes empty", value: DefaultScopeList{}, want: "[]"},
		{name: "scope list empty", value: ScopeList{}, want: "[]"},
		{name: "auth method empty", value: AuthMethodList{}, want: "[]"},
		{name: "connector scopes empty", value: ConnectorScopeList{}, want: "[]"},
		{name: "response types empty", value: ResponseTypeList{}, want: "[]"},
		{name: "response types values", value: ResponseTypeList{ResponseTypeCode, ResponseTypeIDTokenToken}, want: `["code","id_token token"]`},
		{name: "grant types values", value: GrantTypeList{GrantTypeAuthorizationCode}, want: `["authorization_code"]`},
	}
	for _, c := range cases {
		raw, err := json.Marshal(c.value)
		if err != nil {
			t.Fatalf("%s marshal fail: %v", c.name, err)
		}
		if string(raw) != c.want {
			t.Fatalf("%s 期望 %s，实际 %s", c.name, c.want, raw)
		}
	}
}

func TestJSONCarrierStringsHelpers(t *testing.T) {
	if got := (RedirectURIList{"a", "b"}).Strings(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("RedirectURIList.Strings 返回 %v", got)
	}
	if got := (PostLogoutRedirectURIList{"a"}).Strings(); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("PostLogoutRedirectURIList.Strings 返回 %v", got)
	}
	if got := (AllowedOriginList{"a"}).Strings(); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("AllowedOriginList.Strings 返回 %v", got)
	}
	if got := (DefaultScopeList{"a"}).Strings(); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("DefaultScopeList.Strings 返回 %v", got)
	}
	if got := (ScopeList{"a"}).Strings(); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("ScopeList.Strings 返回 %v", got)
	}
	if got := (AuthMethodList{"pwd"}).Strings(); !reflect.DeepEqual(got, []string{"pwd"}) {
		t.Fatalf("AuthMethodList.Strings 返回 %v", got)
	}
	if got := (ConnectorScopeList{ScopeOpenID}).Strings(); !reflect.DeepEqual(got, []string{ScopeOpenID}) {
		t.Fatalf("ConnectorScopeList.Strings 返回 %v", got)
	}
}

func TestConnectorConfigJSONContract(t *testing.T) {
	configType := reflect.TypeOf(ConnectorConfig{})
	for _, forbidden := range []string{"Raw", "Extra"} {
		if _, ok := configType.FieldByName(forbidden); ok {
			t.Fatalf("ConnectorConfig 不应保留逃生舱字段 %s", forbidden)
		}
	}

	raw := []byte(`{"protocol":"oidc","provider":"microsoft","tenant":"contoso","scopes":["openid"],"unknown":"x"}`)
	var config ConnectorConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal fail: %v", err)
	}
	if config.Tenant != "contoso" {
		t.Fatalf("tenant 应提升为一等字段，实际 %q", config.Tenant)
	}
	if config.Protocol != ConnectorProtocolOIDC || config.Provider != ConnectorProviderMicrosoft {
		t.Fatalf("协议/提供商解析错误：%s/%s", config.Protocol, config.Provider)
	}
	if !reflect.DeepEqual(config.Scopes, ConnectorScopeList{ScopeOpenID}) {
		t.Fatalf("scopes 解析错误：%v", config.Scopes)
	}

	out, err := json.Marshal(ConnectorConfig{Protocol: ConnectorProtocolOIDC})
	if err != nil {
		t.Fatalf("marshal fail: %v", err)
	}
	if string(out) != `{"protocol":"oidc","provider":""}` {
		t.Fatalf("空字段应被省略，实际 %s", out)
	}
}

func TestUserIdentityDetailJSONContract(t *testing.T) {
	detailType := reflect.TypeOf(UserIdentityDetail{})
	wantTags := map[string]bool{
		"issuer": true, "subject": true, "email": true, "emailVerified": true,
		"username": true, "displayName": true, "avatarUrl": true,
		"givenName": true, "familyName": true, "locale": true,
	}
	gotTags := make(map[string]bool)
	for i := 0; i < detailType.NumField(); i++ {
		tag := detailType.Field(i).Tag.Get("json")
		gotTags[strings.Split(tag, ",")[0]] = true
	}
	if !reflect.DeepEqual(gotTags, wantTags) {
		t.Fatalf("UserIdentityDetail 白名单字段不一致：%v", gotTags)
	}

	var detail UserIdentityDetail
	if err := json.Unmarshal([]byte(`{"emailVerified":"verified","givenName":"三"}`), &detail); err != nil {
		t.Fatalf("unmarshal fail: %v", err)
	}
	if detail.EmailVerified != EmailVerificationVerified {
		t.Fatalf("emailVerified 应为 %s，实际 %s", EmailVerificationVerified, detail.EmailVerified)
	}
	if detail.GivenName != "三" {
		t.Fatalf("givenName 解析错误：%q", detail.GivenName)
	}
}

func TestConnectorDomainPolicyReusesDomainList(t *testing.T) {
	policyType := reflect.TypeOf(ConnectorDomainPolicy{})
	domainListType := reflect.TypeOf(DomainList(nil))
	for _, fieldName := range []string{"AllowedDomains", "BlockedDomains"} {
		field, ok := policyType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("ConnectorDomainPolicy 缺少字段 %s", fieldName)
		}
		if field.Type != domainListType {
			t.Fatalf("%s 期望 DomainList，实际 %s", fieldName, field.Type)
		}
	}
}
