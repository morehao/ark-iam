package credential

import (
	"testing"
)

// goldenSecretVectors 是凭证摘要的黄金向量：锁定"SHA-256 → 小写 hex、全长不截断"这一契约。
//
// 期望值在本次结构重构**之前**由两个旧实现（重构前的 pkg/iam/apikey.Hash 与
// pkg/token.HashToken，两者实测逐字节相同）产出，属于"改动前事实"的锚点
// ——这两个包已在重构中并入本包，故此处只能以文字记录其出处。任一期望值变化都意味着
// 存量库中已签发的 API Key / OAuth client secret / refresh token 将无法再匹配——
// 那是不可逆的生产事故。
//
// 若确需升级摘要方案（如引入 HMAC + pepper），必须同时给出存量数据的迁移方案，
// 不得只改本表。
var goldenSecretVectors = map[string]string{
	"":    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	"abc": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
	// 下面三条取自改动前旧实现的实测输出。
	"ark-iam": "ed9b4cced3e906213287fbcc3d60de652b1062bb9998021534290593a4973dec",
	"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef": "a8ae6e6ee929abea3afcfc5258c8ccd6f85273e0d4626d26c7279f3250f77c8e",
	"sk_live_9f8e7d6c5b4a3210": "ded96e0027554ce94ef8199450cbbe2f60c84ab929ca492a3f9f718cce304051",
}

// TestHashSecret_Golden 摘要契约的黄金向量测试（等价性验证 L1）。
func TestHashSecret_Golden(t *testing.T) {
	for input, want := range goldenSecretVectors {
		if got := HashSecret(input); got != want {
			t.Fatalf("HashSecret(%q) = %q, want %q —— 摘要契约已漂移，存量凭证将全部失效", input, got, want)
		}
	}
}

// TestGenerateSecret_Shape 高熵机密形态：小写 hex、长度 = 2 × 字节数、批量不重复。
func TestGenerateSecret_Shape(t *testing.T) {
	const rounds = 200
	seen := make(map[string]struct{}, rounds)
	for i := 0; i < rounds; i++ {
		raw, err := GenerateSecret(APIKeyBytes)
		if err != nil {
			t.Fatalf("GenerateSecret fail: %v", err)
		}
		if len(raw) != APIKeyBytes*2 {
			t.Fatalf("expected %d hex chars, got %d (%q)", APIKeyBytes*2, len(raw), raw)
		}
		for _, char := range raw {
			if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
				t.Fatalf("secret %q is not lowercase hex", raw)
			}
		}
		if _, ok := seen[raw]; ok {
			t.Fatalf("duplicate secret generated: %q", raw)
		}
		seen[raw] = struct{}{}
	}
}

// TestGenerateSecret_RejectsNonPositive 非正字节数必须返回错误，不得产出空/零值机密。
func TestGenerateSecret_RejectsNonPositive(t *testing.T) {
	for _, n := range []int{0, -1} {
		raw, err := GenerateSecret(n)
		if err == nil {
			t.Fatalf("GenerateSecret(%d) expected error, got %q", n, raw)
		}
		if raw != "" {
			t.Fatalf("GenerateSecret(%d) must return empty string on error, got %q", n, raw)
		}
	}
}

// TestPrefix 展示前缀：长度由常量给出，短输入原样返回不 panic。
func TestPrefix(t *testing.T) {
	const raw = "0123456789abcdef"
	if got := Prefix(raw, APIKeyPrefixLen); got != raw[:APIKeyPrefixLen] {
		t.Fatalf("Prefix(%q, %d) = %q", raw, APIKeyPrefixLen, got)
	}
	if got := Prefix(raw, ClientSecretPrefixLen); got != raw[:ClientSecretPrefixLen] {
		t.Fatalf("Prefix(%q, %d) = %q", raw, ClientSecretPrefixLen, got)
	}
	const short = "ab"
	if got := Prefix(short, APIKeyPrefixLen); got != short {
		t.Fatalf("Prefix(%q, %d) = %q, want unchanged", short, APIKeyPrefixLen, got)
	}
	if got := Prefix(short, 0); got != short {
		t.Fatalf("Prefix with n=0 must return input unchanged, got %q", got)
	}
}
