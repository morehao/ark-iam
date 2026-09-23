package config

import (
	"os"
	"path/filepath"
	"testing"
)

// 引导令牌的 YAML 键名必须与 struct tag 严格对应：写错（`bootstrap_token` vs
// `bootstrapToken`）不会报错，只会静默解析成空串，然后表现为"配置了却仍返回
// 503/107002"——这类问题在本地很难自查。因此直接拿**本应用真实的 config.yaml**
// 过一遍真实加载器，把键名绑定钉死。
func TestLoadConfigBindsInstallBootstrapToken(t *testing.T) {
	path := filepath.Join("config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("配置文件不存在: %v", err)
	}

	LoadConfig(path)

	if Conf == nil {
		t.Fatal("LoadConfig 后 Conf 不应为 nil")
	}
	if got := Conf.Install.BootstrapToken; got == "" {
		t.Fatalf("install.bootstrapToken 未解析出值（键名与 struct tag 不匹配？）: %q", got)
	}
	if Conf.Install.BootstrapToken != "dev-bootstrap-token" {
		t.Fatalf("install.bootstrapToken = %q, want %q", Conf.Install.BootstrapToken, "dev-bootstrap-token")
	}
}
