package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestOIDCConsolesParsesCompleteAddresses 控制台部署地址是**完整 URL**，而不是"根地址 + 代码拼路径"。
//
// 这条配置的意义：初始化页面回显给运维的地址与 L1 引导写进回调白名单的地址必须同源，
// 否则会出现"页面说稍后从 A 登录、实际只允许 B 回调"的错配（表现为登录后回调被拒）。
func TestOIDCConsolesParsesCompleteAddresses(t *testing.T) {
	const raw = `
issuer: "https://iam.example.com/oidc"
frontendLoginURL: "https://login.example.com/login"
consoles:
  platformAdminWeb:
    redirectURIs:
      - "https://admin.example.com/auth/callback"
      - "https://admin.example.com/silent-callback"
    postLogoutRedirectURIs:
      - "https://admin.example.com/login"
    backChannelLogoutURI: "https://iam.example.com/oidc/bc-logout/platform"
  tenantAdminWeb:
    redirectURIs:
      - "https://tenant.example.com/auth/callback"
    postLogoutRedirectURIs:
      - "https://tenant.example.com/login"
    backChannelLogoutURI: ""
`
	var oidc OIDC
	if err := yaml.Unmarshal([]byte(raw), &oidc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := oidc.Consoles.PlatformAdminWeb.RedirectURIs; len(got) != 2 || got[0] != "https://admin.example.com/auth/callback" {
		t.Errorf("platformAdminWeb.redirectURIs = %v", got)
	}
	if got := oidc.Consoles.PlatformAdminWeb.PostLogoutRedirectURIs; len(got) != 1 || got[0] != "https://admin.example.com/login" {
		t.Errorf("platformAdminWeb.postLogoutRedirectURIs = %v", got)
	}
	if got := oidc.Consoles.PlatformAdminWeb.BackChannelLogoutURI; got != "https://iam.example.com/oidc/bc-logout/platform" {
		t.Errorf("platformAdminWeb.backChannelLogoutURI = %q", got)
	}
	if got := oidc.Consoles.TenantAdminWeb.RedirectURIs; len(got) != 1 || got[0] != "https://tenant.example.com/auth/callback" {
		t.Errorf("tenantAdminWeb.redirectURIs = %v", got)
	}
	// 留空即"按 issuer 派生"：配置层不做任何补全，补全归 pkg/seed.withDefaults，
	// 保证"配置没写"与"写空"语义一致。
	if got := oidc.Consoles.TenantAdminWeb.BackChannelLogoutURI; got != "" {
		t.Errorf("留空的 backChannelLogoutURI 必须保持空（由 L1 按 issuer 派生），实际 %q", got)
	}
}

// TestOIDCConsolesAbsentIsZeroValue 未配置 consoles 时必须是零值。
//
// 这条是"零行为变更"的保证：零值进入 L1 的 withDefaults 后回落到内置缺省地址
// （本地开发端口 4001/4002），与改造前硬编码在 pkg/seed 里的值逐字节相同；
// 配置层不得自行填默认值，否则"缺省值"会分裂成两份、早晚不一致。
func TestOIDCConsolesAbsentIsZeroValue(t *testing.T) {
	var oidc OIDC
	if err := yaml.Unmarshal([]byte("issuer: \"http://localhost:8100/oidc\"\n"), &oidc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(oidc.Consoles.PlatformAdminWeb.RedirectURIs) != 0 ||
		len(oidc.Consoles.PlatformAdminWeb.PostLogoutRedirectURIs) != 0 ||
		oidc.Consoles.PlatformAdminWeb.BackChannelLogoutURI != "" {
		t.Errorf("未配置 consoles 时必须为零值，实际 %+v", oidc.Consoles.PlatformAdminWeb)
	}
	if len(oidc.Consoles.TenantAdminWeb.RedirectURIs) != 0 {
		t.Errorf("未配置 consoles 时必须为零值，实际 %+v", oidc.Consoles.TenantAdminWeb)
	}
}

// TestShippedConfigsParseConsoles 随仓库发布的 app 配置必须能解析出 consoles 段。
//
// 配置文件的语法错误只会在进程启动时暴露（且 app 启动需要 DB/Redis 等环境），
// 因此在这里直接把真实文件解析一遍——它能在 CI 里拦住"改配置改坏了"。
func TestShippedConfigsParseConsoles(t *testing.T) {
	for _, path := range []string{
		"../../apps/auth/config/config.yaml",
		"../../apps/gateway/config/config.yaml",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		got := cfg.OIDC.Consoles.PlatformAdminWeb
		if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "http://localhost:4001/auth/callback" {
			t.Errorf("%s: platformAdminWeb.redirectURIs = %v", path, got.RedirectURIs)
		}
		if len(got.PostLogoutRedirectURIs) != 1 || got.PostLogoutRedirectURIs[0] != "http://localhost:4001/login" {
			t.Errorf("%s: platformAdminWeb.postLogoutRedirectURIs = %v", path, got.PostLogoutRedirectURIs)
		}
		tenant := cfg.OIDC.Consoles.TenantAdminWeb
		if len(tenant.RedirectURIs) != 1 || tenant.RedirectURIs[0] != "http://localhost:4002/auth/callback" {
			t.Errorf("%s: tenantAdminWeb.redirectURIs = %v", path, tenant.RedirectURIs)
		}
		// backChannelLogoutURI 留空 → 由 L1 按该部署的 issuer 派生（auth 8081 / gateway 8100 不同）
		if got.BackChannelLogoutURI != "" {
			t.Errorf("%s: backChannelLogoutURI 应留空由 issuer 派生，实际 %q", path, got.BackChannelLogoutURI)
		}
	}
}
