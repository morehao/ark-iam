package svcoidc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
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

	if _, err := loadSigningKeys(); err == nil {
		t.Fatal("expected loadSigningKeys to fail in non-dev without configured key")
	}
	if _, _, err := loadEncryptionKey(); err == nil {
		t.Fatal("expected loadEncryptionKey to fail in non-dev without configured key")
	}

	// dev 环境允许临时/测试密钥
	appconfig.Conf = &pkgconfig.Config{
		Server: pkgconfig.Server{Env: "dev"},
		OIDC:   pkgconfig.OIDC{},
	}
	if _, err := loadSigningKeys(); err != nil {
		t.Fatalf("expected loadSigningKeys to succeed in dev, got %v", err)
	}
	if _, _, err := loadEncryptionKey(); err != nil {
		t.Fatalf("expected loadEncryptionKey to succeed in dev, got %v", err)
	}
}

// TestLoadSigningKeysMultiKey 覆盖最小多 key：
//   - keys 列表里恰好一个 active；全部公钥都要发布（含过渡期旧 key）；
//   - 未显式配置 kid 时按公钥派生稳定 kid；
//   - 0 个或多个 active、kid 重复都是配置错误（fail-closed，不猜测）。
func TestLoadSigningKeysMultiKey(t *testing.T) {
	dir := t.TempDir()
	oldPEM, oldKid := writeTestKeyFile(t, dir, "old.pem")
	newPEM, newKid := writeTestKeyFile(t, dir, "new.pem")

	cfg := &pkgconfig.OIDC{
		Keys: []pkgconfig.SigningKeyConfig{
			{Kid: oldKid, PrivateKeyPEM: oldPEM},
			{Kid: newKid, PrivateKeyPEM: newPEM, Active: true},
		},
	}
	loaded, err := loadSigningKeysFromConfig(cfg)
	if err != nil {
		t.Fatalf("loadSigningKeysFromConfig failed: %v", err)
	}
	if loaded.ActiveKeyID != newKid {
		t.Fatalf("expected active kid %q, got %q", newKid, loaded.ActiveKeyID)
	}
	if len(loaded.Published) != 2 {
		t.Fatalf("expected 2 published keys (active + transitional old), got %d", len(loaded.Published))
	}
	for _, kid := range []string{oldKid, newKid} {
		if loaded.Published[kid] == nil {
			t.Fatalf("expected published key %q", kid)
		}
	}

	// 未配置 kid：由公钥派生，且两次调用结果一致（可复现）。
	derived, err := loadSigningKeysFromConfig(&pkgconfig.OIDC{
		Keys: []pkgconfig.SigningKeyConfig{{PrivateKeyPEM: newPEM, Active: true}},
	})
	if err != nil {
		t.Fatalf("loadSigningKeysFromConfig (derived kid) failed: %v", err)
	}
	if derived.ActiveKeyID == "" {
		t.Fatal("expected derived kid to be non-empty")
	}
	if DeriveKeyID(&derived.ActiveKey.PublicKey) != derived.ActiveKeyID {
		t.Fatal("derived kid must be stable across calls")
	}

	// kid 重复：配置错误
	if _, err := loadSigningKeysFromConfig(&pkgconfig.OIDC{
		Keys: []pkgconfig.SigningKeyConfig{
			{Kid: "dup", PrivateKeyPEM: oldPEM, Active: true},
			{Kid: "dup", PrivateKeyPEM: newPEM},
		},
	}); err == nil {
		t.Fatal("expected duplicate kid to be rejected")
	}

	// 没有 active：配置错误
	if _, err := loadSigningKeysFromConfig(&pkgconfig.OIDC{
		Keys: []pkgconfig.SigningKeyConfig{{Kid: "k1", PrivateKeyPEM: oldPEM}},
	}); err == nil {
		t.Fatal("expected missing active key to be rejected")
	}

	// 多个 active：配置错误
	if _, err := loadSigningKeysFromConfig(&pkgconfig.OIDC{
		Keys: []pkgconfig.SigningKeyConfig{
			{Kid: "k1", PrivateKeyPEM: oldPEM, Active: true},
			{Kid: "k2", PrivateKeyPEM: newPEM, Active: true},
		},
	}); err == nil {
		t.Fatal("expected multiple active keys to be rejected")
	}
}

// TestLoadSigningKeysLegacySingleKey 单 key 三元组经 EffectiveSigningKeys 归一后
// 仍然可用（零破坏迁移）。
func TestLoadSigningKeysLegacySingleKey(t *testing.T) {
	dir := t.TempDir()
	pemData, kid := writeTestKeyFile(t, dir, "legacy.pem")

	loaded, err := loadSigningKeysFromConfig(&pkgconfig.OIDC{
		SigningKeyID:          kid,
		SigningPrivateKeyPath: writeFile(t, dir, "legacy-path.pem", pemData),
	})
	if err != nil {
		t.Fatalf("loadSigningKeysFromConfig failed: %v", err)
	}
	if loaded.ActiveKeyID != kid {
		t.Fatalf("expected %q, got %q", kid, loaded.ActiveKeyID)
	}
	if len(loaded.Published) != 1 || loaded.Published[kid] == nil {
		t.Fatalf("expected exactly one published key %q, got %v", kid, loaded.Published)
	}
}

// TestDeriveKeyID_CrossPackageGolden 锁定 kid 派生算法的跨包 golden。
//
// pkg/oidckit 为 RP 侧复用同一算法另存了一份实现（pkg 不能反向依赖 apps），
// 两侧测试都断言这同一个字面量：pkg/oidckit/keys_test.go 的 TestDeriveKeyID_Golden
// 与本用例。任一侧改了截断长度/编码方式，另一侧立刻失败——kid 与 OP 实际签发的
// 不一致会导致全量请求 401，且症状只在运行时出现，必须靠测试兜住。
func TestDeriveKeyID_CrossPackageGolden(t *testing.T) {
	const modulusHex = "c3530e0d1a5d0b5f2e7c4a9b8d6f3c2e1a4b7d9f0c2e5a8b1d4f7c0e3a6b9d2c"
	n, ok := new(big.Int).SetString(modulusHex, 16)
	if !ok {
		t.Fatalf("invalid modulus hex")
	}
	if got := DeriveKeyID(&rsa.PublicKey{N: n, E: 65537}); got != "DcYoXzsKCpT1wWLw_BYXaA" {
		t.Fatalf("kid derivation drifted from pkg/oidckit golden: got %q, want %q", got, "DcYoXzsKCpT1wWLw_BYXaA")
	}
}

func writeTestKeyFile(t *testing.T, dir, name string) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	encoded := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	kid := DeriveKeyID(&key.PublicKey)
	return string(encoded), kid
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}
