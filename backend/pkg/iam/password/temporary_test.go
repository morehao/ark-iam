package password

import (
	"strings"
	"testing"
)

// TestGenerateTemporary_MeetsStrength 临时密码必须必然通过强度校验与长度约定。
func TestGenerateTemporary_MeetsStrength(t *testing.T) {
	for i := 0; i < 500; i++ {
		generated, err := GenerateTemporary()
		if err != nil {
			t.Fatalf("GenerateTemporary fail: %v", err)
		}
		if len(generated) != temporaryLength {
			t.Fatalf("expected length %d, got %d (%q)", temporaryLength, len(generated), generated)
		}
		if err := ValidateStrength(generated); err != nil {
			t.Fatalf("generated password %q fails ValidateStrength: %v", generated, err)
		}
	}
}

// TestGenerateTemporary_ExcludesAmbiguousChars 生成结果不得包含易混淆字符（0/1/l/I/O）。
func TestGenerateTemporary_ExcludesAmbiguousChars(t *testing.T) {
	const ambiguous = "01lIO"
	for i := 0; i < 500; i++ {
		generated, err := GenerateTemporary()
		if err != nil {
			t.Fatalf("GenerateTemporary fail: %v", err)
		}
		if strings.ContainsAny(generated, ambiguous) {
			t.Fatalf("generated password %q contains ambiguous char", generated)
		}
	}
}

// TestGenerateTemporary_AlwaysHasEachClass 前三位强制各类字符，且洗牌后仍然都存在。
func TestGenerateTemporary_AlwaysHasEachClass(t *testing.T) {
	for i := 0; i < 500; i++ {
		generated, err := GenerateTemporary()
		if err != nil {
			t.Fatalf("GenerateTemporary fail: %v", err)
		}
		if !strings.ContainsAny(generated, lowerChars) {
			t.Fatalf("missing lowercase: %q", generated)
		}
		if !strings.ContainsAny(generated, upperChars) {
			t.Fatalf("missing uppercase: %q", generated)
		}
		if !strings.ContainsAny(generated, digitChars) {
			t.Fatalf("missing digit: %q", generated)
		}
	}
}

// TestGenerateTemporary_Unique 批量生成不得出现重复（16 位 × 57 字符集，碰撞概率可忽略）。
func TestGenerateTemporary_Unique(t *testing.T) {
	const rounds = 10000
	seen := make(map[string]struct{}, rounds)
	for i := 0; i < rounds; i++ {
		generated, err := GenerateTemporary()
		if err != nil {
			t.Fatalf("GenerateTemporary fail: %v", err)
		}
		if _, ok := seen[generated]; ok {
			t.Fatalf("duplicate temporary password generated: %q", generated)
		}
		seen[generated] = struct{}{}
	}
	if len(seen) != rounds {
		t.Fatalf("expected %d unique passwords, got %d", rounds, len(seen))
	}
}

// TestBootstrapAdminPassword_Stable 种子 bootstrap 口令是部署文档与 e2e 的稳定契约：
// 值一旦变化必须同步 e2e/config.ts 与部署文档，因此在此锁定。
func TestBootstrapAdminPassword_Stable(t *testing.T) {
	if BootstrapAdminPassword != "admin123" {
		t.Fatalf("bootstrap admin password changed to %q; update e2e/config.ts and deploy docs together", BootstrapAdminPassword)
	}
	// bootstrap 口令是唯一固定默认口令，绝不能由生成器产出（否则等于把它变成通用临时口令）。
	for i := 0; i < 200; i++ {
		generated, err := GenerateTemporary()
		if err != nil {
			t.Fatalf("GenerateTemporary fail: %v", err)
		}
		if generated == BootstrapAdminPassword {
			t.Fatalf("temporary password equals bootstrap password")
		}
	}
}
