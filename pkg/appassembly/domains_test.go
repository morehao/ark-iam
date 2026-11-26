package appassembly

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
	for _, must := range []Domain{DomainCoreAuth, DomainInstall, DomainUser, DomainDepartment, DomainRole, DomainMenu, DomainApplication} {
		if !set[must] {
			t.Fatalf("DefaultConsoleDomains missing %q", must)
		}
	}
}

// TestDomainDependency 校验域依赖：只选 role 不选 menu 应报错；menu 依赖 application。
func TestDomainDependency(t *testing.T) {
	_, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}, Domains: []Domain{DomainRole}})
	var depErr *ErrDomainDependency
	if !errors.As(err, &depErr) {
		t.Fatalf("New(role only): want ErrDomainDependency, got %v", err)
	}
	// role + menu 仍缺 application（role→menu→application 传递）应报错。
	if _, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}, Domains: []Domain{DomainRole, DomainMenu}}); !errors.As(err, &depErr) {
		t.Fatalf("New(role+menu): want ErrDomainDependency (menu requires application), got %v", err)
	}
	// role + menu + application 应通过（依赖闭包闭合）。
	if _, err := New(Config{DB: DBConfig{DSN: "sqlite://:memory:"}, Domains: []Domain{DomainRole, DomainMenu, DomainApplication}}); err != nil {
		t.Fatalf("New(role+menu+application): unexpected %v", err)
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
