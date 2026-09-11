package tenant

import (
	"regexp"
	"testing"
)

// TestGenerateCode 校验编码形态：t_<12 位小写 hex>。
func TestGenerateCode(t *testing.T) {
	code, err := GenerateCode()
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	pattern := regexp.MustCompile(`^t_[0-9a-f]{12}$`)
	if !pattern.MatchString(code) {
		t.Errorf("GenerateCode = %q, want match %s", code, pattern.String())
	}
}

// TestGenerateCodeUnique 校验随机段保证编码不重复。
func TestGenerateCodeUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatalf("GenerateCode failed: %v", err)
		}
		if _, ok := seen[code]; ok {
			t.Fatalf("GenerateCode duplicate: %s", code)
		}
		seen[code] = struct{}{}
	}
}
