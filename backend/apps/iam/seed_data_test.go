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
}
