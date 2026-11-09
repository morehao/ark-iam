package credential

import (
	"strings"
	"testing"
)

// TestGenerateTemporaryPassword_MeetsStrength 临时密码必须必然通过强度校验与长度约定。
func TestGenerateTemporaryPassword_MeetsStrength(t *testing.T) {
	for i := 0; i < 500; i++ {
		generated, err := GenerateTemporaryPassword()
		if err != nil {
			t.Fatalf("GenerateTemporaryPassword fail: %v", err)
		}
		if len(generated) != temporaryLength {
			t.Fatalf("expected length %d, got %d (%q)", temporaryLength, len(generated), generated)
		}
		if err := ValidateStrength(generated); err != nil {
			t.Fatalf("generated password %q fails ValidateStrength: %v", generated, err)
		}
	}
}

// TestGenerateTemporaryPassword_ExcludesAmbiguousChars 生成结果不得包含易混淆字符（0/1/l/I/O）。
func TestGenerateTemporaryPassword_ExcludesAmbiguousChars(t *testing.T) {
	const ambiguous = "01lIO"
	for i := 0; i < 500; i++ {
		generated, err := GenerateTemporaryPassword()
		if err != nil {
			t.Fatalf("GenerateTemporaryPassword fail: %v", err)
		}
		if strings.ContainsAny(generated, ambiguous) {
			t.Fatalf("generated password %q contains ambiguous char", generated)
		}
	}
}

// TestGenerateTemporaryPassword_AlwaysHasEachClass 前三位强制各类字符，且洗牌后仍然都存在。
func TestGenerateTemporaryPassword_AlwaysHasEachClass(t *testing.T) {
	for i := 0; i < 500; i++ {
		generated, err := GenerateTemporaryPassword()
		if err != nil {
			t.Fatalf("GenerateTemporaryPassword fail: %v", err)
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

// TestGenerateTemporaryPassword_Unique 批量生成不得出现重复（16 位 × 57 字符集，碰撞概率可忽略）。
func TestGenerateTemporaryPassword_Unique(t *testing.T) {
	const rounds = 10000
	seen := make(map[string]struct{}, rounds)
	for i := 0; i < rounds; i++ {
		generated, err := GenerateTemporaryPassword()
		if err != nil {
			t.Fatalf("GenerateTemporaryPassword fail: %v", err)
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
