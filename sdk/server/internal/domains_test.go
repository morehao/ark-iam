package internal

import (
	"errors"
	"testing"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"

	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

func TestResolveDomains(t *testing.T) {
	if got := resolveDomains(nil); len(got) != len(allDomains) {
		t.Fatalf("resolveDomains(nil): want all %d domains, got %d", len(allDomains), len(got))
	}
	// 去重
	got := resolveDomains([]Domain{DomainUser, DomainUser, DomainRole})
	if len(got) != 2 {
		t.Fatalf("resolveDomains dedupe: want 2, got %d (%v)", len(got), got)
	}
	// DefaultConsoleDomains 应包含核心域
	set := make(map[Domain]bool)
	for _, d := range DefaultConsoleDomains {
		set[d] = true
	}
	for _, must := range []Domain{DomainCoreAuth, DomainInstall, DomainUser, DomainDepartment, DomainRole, DomainMenu} {
		if !set[must] {
			t.Fatalf("DefaultConsoleDomains missing %q", must)
		}
	}
}

// TestDomainDependency 校验域依赖：只选 role 不选 menu 应报错。
func TestDomainDependency(t *testing.T) {
	_, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}, Domains: []Domain{DomainRole}})
	var depErr *ErrDomainDependency
	if !errors.As(err, &depErr) {
		t.Fatalf("New(role only): want ErrDomainDependency, got %v", err)
	}
	// role + menu 应通过。
	if _, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}, Domains: []Domain{DomainRole, DomainMenu}}); err != nil {
		t.Fatalf("New(role+menu): unexpected %v", err)
	}
}

// TestNonDevSecurityValidation 校验 prod 缺安全配置被拒绝。
func TestNonDevSecurityValidation(t *testing.T) {
	_, err := New(Config{
		Env: "prod",
		DB:  DBConfig{DSN: "sqlite://:memory:"},
		Raw: &pkconfig.Config{}, // 缺 cookieSecure/密钥/encryptionKey
	})
	if err == nil {
		t.Fatal("New(prod, insecure config): want error, got nil")
	}
}
