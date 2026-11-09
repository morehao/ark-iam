package seed

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
)

// TestBackChannelLogoutURI 钉住 bc-logout 地址的派生规则。
//
// 这是"配置化不能改变现有部署行为"的核心断言：默认 issuer 必须派生出现值与
// application_client.back_channel_logout_uri 完全一致的地址（含路径前缀、无重复 /oidc）。
func TestBackChannelLogoutURI(t *testing.T) {
	cases := []struct {
		name   string
		issuer string
		path   string
		want   string
	}{
		{
			name:   "默认 issuer 派生平台接收端（与改造前硬编码值一致）",
			issuer: "http://localhost:8100/oidc",
			path:   model.SeedBackChannelLogoutPathPlatform,
			want:   "http://localhost:8100/oidc/bc-logout/platform",
		},
		{
			name:   "默认 issuer 派生租户接收端",
			issuer: "http://localhost:8100/oidc",
			path:   model.SeedBackChannelLogoutPathTenant,
			want:   "http://localhost:8100/oidc/bc-logout/tenant",
		},
		{
			name:   "issuer 带尾斜杠不产生双斜杠",
			issuer: "https://sso.acme.com/oidc/",
			path:   model.SeedBackChannelLogoutPathPlatform,
			want:   "https://sso.acme.com/oidc/bc-logout/platform",
		},
		{
			name:   "issuer 无路径时直接拼接",
			issuer: "https://sso.acme.com",
			path:   model.SeedBackChannelLogoutPathTenant,
			want:   "https://sso.acme.com/bc-logout/tenant",
		},
		{
			name:   "path 无前导斜杠时补齐",
			issuer: "https://sso.acme.com/oidc",
			path:   "bc-logout/platform",
			want:   "https://sso.acme.com/oidc/bc-logout/platform",
		},
		{
			name:   "path 为空即返回归一后的 issuer（不凭空拼地址）",
			issuer: "https://sso.acme.com/oidc/",
			path:   "",
			want:   "https://sso.acme.com/oidc",
		},
		{
			name:   "issuer 首尾空白被裁剪",
			issuer: " https://sso.acme.com/oidc ",
			path:   model.SeedBackChannelLogoutPathPlatform,
			want:   "https://sso.acme.com/oidc/bc-logout/platform",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := backChannelLogoutURI(c.issuer, c.path); got != c.want {
				t.Errorf("backChannelLogoutURI(%q, %q) = %q, want %q", c.issuer, c.path, got, c.want)
			}
		})
	}
}

// TestWithDefaults 钉住 Definition 的零值回落规则。
//
// 切片的 nil / 非 nil 空切片语义必须区分：nil 是"没配"（回落 dev 缺省），
// 空切片是"显式配成空"（不回落）——否则运维无法表达"这个客户端不开放任何回调"。
func TestWithDefaults(t *testing.T) {
	t.Run("零值全部回落内置缺省", func(t *testing.T) {
		got := withDefaults(Definition{})
		if got.TenantName != defaultTenantName {
			t.Errorf("TenantName = %q, want %q", got.TenantName, defaultTenantName)
		}
		if got.AdminUsername != defaultAdminUsername || got.AdminName != defaultAdminName {
			t.Errorf("admin identity = %q/%q, want %q/%q", got.AdminUsername, got.AdminName, defaultAdminUsername, defaultAdminName)
		}
		if got.AdminEmail != defaultAdminEmail || got.AdminPhone != defaultAdminPhone {
			t.Errorf("admin contact = %q/%q, want %q/%q", got.AdminEmail, got.AdminPhone, defaultAdminEmail, defaultAdminPhone)
		}
		if got.AdminPasswordStatus != model.PasswordStatusNormal {
			t.Errorf("AdminPasswordStatus = %q, want %q", got.AdminPasswordStatus, model.PasswordStatusNormal)
		}
		if got.Issuer != defaultIssuer {
			t.Errorf("Issuer = %q, want %q", got.Issuer, defaultIssuer)
		}
		if len(got.Consoles.PlatformAdminWeb.RedirectURIs) != 1 || got.Consoles.PlatformAdminWeb.RedirectURIs[0] != defaultConsolePlatformRedirectURI {
			t.Errorf("platform redirect = %v, want [%s]", got.Consoles.PlatformAdminWeb.RedirectURIs, defaultConsolePlatformRedirectURI)
		}
		if len(got.Consoles.TenantAdminWeb.PostLogoutRedirectURIs) != 1 || got.Consoles.TenantAdminWeb.PostLogoutRedirectURIs[0] != defaultConsoleTenantPostLogoutURI {
			t.Errorf("tenant post-logout = %v, want [%s]", got.Consoles.TenantAdminWeb.PostLogoutRedirectURIs, defaultConsoleTenantPostLogoutURI)
		}
		if got.Consoles.PlatformAdminWeb.BackChannelLogoutURI != "http://localhost:8100/oidc/bc-logout/platform" {
			t.Errorf("platform bc-logout = %q", got.Consoles.PlatformAdminWeb.BackChannelLogoutURI)
		}
		if got.Consoles.TenantAdminWeb.BackChannelLogoutURI != "http://localhost:8100/oidc/bc-logout/tenant" {
			t.Errorf("tenant bc-logout = %q", got.Consoles.TenantAdminWeb.BackChannelLogoutURI)
		}
	})

	t.Run("自定义 issuer 参与 bc-logout 派生", func(t *testing.T) {
		got := withDefaults(Definition{Issuer: "https://sso.acme.com/oidc"})
		if got.Consoles.PlatformAdminWeb.BackChannelLogoutURI != "https://sso.acme.com/oidc/bc-logout/platform" {
			t.Errorf("platform bc-logout = %q", got.Consoles.PlatformAdminWeb.BackChannelLogoutURI)
		}
		if got.Consoles.TenantAdminWeb.BackChannelLogoutURI != "https://sso.acme.com/oidc/bc-logout/tenant" {
			t.Errorf("tenant bc-logout = %q", got.Consoles.TenantAdminWeb.BackChannelLogoutURI)
		}
	})

	t.Run("非 nil 空切片表达显式配置为空", func(t *testing.T) {
		got := withDefaults(Definition{Consoles: ConsolesConfig{
			PlatformAdminWeb: ConsoleRedirects{RedirectURIs: []string{}},
		}})
		if got.Consoles.PlatformAdminWeb.RedirectURIs == nil {
			t.Fatal("显式空切片被回落成了缺省值：运维无法表达「不开放任何回调」")
		}
		if len(got.Consoles.PlatformAdminWeb.RedirectURIs) != 0 {
			t.Errorf("RedirectURIs = %v, want empty", got.Consoles.PlatformAdminWeb.RedirectURIs)
		}
		// 未显式配置的兄弟字段仍回落缺省
		if len(got.Consoles.PlatformAdminWeb.PostLogoutRedirectURIs) != 1 {
			t.Errorf("未配置的 post-logout 未回落缺省: %v", got.Consoles.PlatformAdminWeb.PostLogoutRedirectURIs)
		}
	})

	t.Run("显式 bc-logout 覆盖优先于派生", func(t *testing.T) {
		got := withDefaults(Definition{
			Issuer: "https://sso.acme.com/oidc",
			Consoles: ConsolesConfig{
				PlatformAdminWeb: ConsoleRedirects{BackChannelLogoutURI: "https://admin-rcv.acme.com/bc"},
			},
		})
		if got.Consoles.PlatformAdminWeb.BackChannelLogoutURI != "https://admin-rcv.acme.com/bc" {
			t.Errorf("platform bc-logout = %q, want 覆盖值", got.Consoles.PlatformAdminWeb.BackChannelLogoutURI)
		}
		if got.Consoles.TenantAdminWeb.BackChannelLogoutURI != "https://sso.acme.com/oidc/bc-logout/tenant" {
			t.Errorf("未覆盖的一侧应走派生, got %q", got.Consoles.TenantAdminWeb.BackChannelLogoutURI)
		}
	})
}
