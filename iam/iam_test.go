package iam

import (
	"testing"

	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"
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
