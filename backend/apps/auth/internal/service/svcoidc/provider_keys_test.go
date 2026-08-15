package svcoidc

import (
	"testing"

	appconfig "github.com/morehao/ark-iam/auth/config"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
)

// TestLoadKeysFailClosedInNonDev 覆盖 H4：
// 非 dev 环境未配置签名/加密密钥时必须启动失败（fail-closed），禁止使用临时/测试密钥。
func TestLoadKeysFailClosedInNonDev(t *testing.T) {
	prev := appconfig.Conf
	defer func() { appconfig.Conf = prev }()
	appconfig.Conf = &pkgconfig.Config{
		Server: pkgconfig.Server{Env: "prod"},
		OIDC:   pkgconfig.OIDC{},
	}

	if _, _, err := loadSigningKey(); err == nil {
		t.Fatal("expected loadSigningKey to fail in non-dev without configured key")
	}
	if _, _, err := loadEncryptionKey(); err == nil {
		t.Fatal("expected loadEncryptionKey to fail in non-dev without configured key")
	}

	// dev 环境允许临时/测试密钥
	appconfig.Conf = &pkgconfig.Config{
		Server: pkgconfig.Server{Env: "dev"},
		OIDC:   pkgconfig.OIDC{},
	}
	if _, _, err := loadSigningKey(); err != nil {
		t.Fatalf("expected loadSigningKey to succeed in dev, got %v", err)
	}
	if _, _, err := loadEncryptionKey(); err != nil {
		t.Fatalf("expected loadEncryptionKey to succeed in dev, got %v", err)
	}
}
