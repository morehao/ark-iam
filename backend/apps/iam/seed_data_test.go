package iam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedDataIncludesPostLogoutRedirectForTestRPClient(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "scripts", "sql", "iam_seed_data.sql"))
	if err != nil {
		t.Fatalf("read iam_seed_data.sql: %v", err)
	}

	seedSQL := string(content)
	if !strings.Contains(seedSQL, "'test-rp-client'") {
		t.Fatal("expected test-rp-client seed data")
	}
	if !strings.Contains(seedSQL, "http://localhost:3001/login") {
		t.Fatal("expected test-rp-client to include post logout redirect uri")
	}
	if !strings.Contains(seedSQL, "'platform-admin-web'") {
		t.Fatal("expected platform-admin-web seed data")
	}
	if !strings.Contains(seedSQL, "'platform-admin-web', 'IAM管理平台', '[\"http://localhost:3000/auth/callback\"]', '[\"authorization_code\",\"refresh_token\"]', '[\"code\"]', 'none', 1, '[\"openid\",\"profile\",\"email\"]', '[\"http://localhost:3000/login\"]', 'first_party', 0, 'enable', 1, 0, 0, 0") {
		t.Fatal("expected platform-admin-web client to be marked as is_system=1")
	}
}
