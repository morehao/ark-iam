package model

import (
	"sort"
	"testing"
)

// TestSeedAuthorityMatrixIsUniqueAndWellFormed 矩阵自身的不变式：
// (entity, field) 唯一、mode 合法、键非空——矩阵是种子与控制台的共同真相源，声明错误会静默改变写者。
func TestSeedAuthorityMatrixIsUniqueAndWellFormed(t *testing.T) {
	allowed := map[SeedFieldMode]bool{
		SeedFieldImmutable:  true,
		SeedFieldCreateOnly: true,
	}
	seen := map[[2]string]bool{}
	for _, authority := range SeedFieldAuthorities {
		if authority.Entity == "" || authority.Field == "" {
			t.Fatalf("authority 键不得为空: %+v", authority)
		}
		if !allowed[authority.Mode] {
			t.Fatalf("未知语义 %q: %s.%s", authority.Mode, authority.Entity, authority.Field)
		}
		key := [2]string{authority.Entity, authority.Field}
		if seen[key] {
			t.Fatalf("重复声明: %s.%s", authority.Entity, authority.Field)
		}
		seen[key] = true
	}
}

// TestSeedFieldImmutableOnlyForHardReasons 只有「被控制台改写后会导致鉴权被绕过或控制台自我锁死」
// 的字段才是 immutable：安全不变式（source/admin_type/平台租户 status）与身份编码（内置应用/客户端 code）。
// 展示字段一律 create_only（归运维），未声明字段默认 create_only。
func TestSeedFieldImmutableOnlyForHardReasons(t *testing.T) {
	if !SeedFieldIsImmutable(SeedEntityApplication, "source") {
		t.Error("application.source 是安全不变式，应 immutable")
	}
	if !SeedFieldIsImmutable(SeedEntityApplication, "code") {
		t.Error("application.code 是各控制台菜单入口的定位值，改名会锁死控制台，应 immutable")
	}
	if !SeedFieldIsImmutable(SeedEntityApplicationClient, "code") {
		t.Error("application_client.code 是网关 aud 白名单取值来源，应 immutable")
	}
	for _, field := range []string{"name", "description", "status", "sort", "logo_url", "homepage_url"} {
		if SeedFieldIsImmutable(SeedEntityApplication, field) {
			t.Errorf("application.%s 属展示/运行参数（create_only，归运维），不应 immutable", field)
		}
	}
	if SeedFieldIsImmutable(SeedEntityApplication, "undeclared_field") {
		t.Error("未声明字段应默认为 create_only（归运维）")
	}
	// 菜单：**没有任何 immutable 字段**——L1 按 seed_key（种子身份键，非控制台字段）认行，
	// 于是 code/app_id 连同展示与结构字段一律归运维可改。
	for _, field := range []string{"seed_key", "app_id", "code", "name", "path", "icon", "sort", "component", "type", "visibility", "parent_id", "status"} {
		if SeedFieldIsImmutable(SeedEntityMenu, field) {
			t.Errorf("menu.%s 归运维（create_only），不应 immutable", field)
		}
	}
	// 一次性改名（原 migrate_once）已随启动期播种删除，逐版本改名不再是种子职责
	if got := SeedFieldModeOf(SeedEntityTenant, "name"); got != SeedFieldCreateOnly {
		t.Errorf("tenant.name mode = %q, want %q（改名归运维，不再由种子跨版本迁移）", got, SeedFieldCreateOnly)
	}
	if got := SeedFieldModeOf(SeedEntityDepartment, "name"); got != SeedFieldCreateOnly {
		t.Errorf("department.name mode = %q, want %q", got, SeedFieldCreateOnly)
	}
}

// TestSeedImmutableFieldsPinned 钉住不可变字段的完整集合。
//
// 这是本改动后矩阵的主要作用：它不再驱动任何写入（L1 只创建、不收敛），而是**声明**——
// 一旦这组集合发生变化，必然是有人有意改动了"哪些内置字段控制台必须拒写"，需要同步核对
// 控制台侧的拒写点（见 docs/design/system-design.md §4.5），因此用测试钉住、强制 review。
func TestSeedImmutableFieldsPinned(t *testing.T) {
	want := map[string][]string{
		SeedEntityTenant:            {"status"},
		SeedEntityApplication:       {"code", "source"},
		SeedEntityApplicationClient: {"code", "source"},
		SeedEntityRole:              {"admin_type"},
		SeedEntityUser:              {"source"},
	}
	for entity, wantFields := range want {
		got := SeedImmutableFields(entity)
		sort.Strings(got)
		sorted := append([]string(nil), wantFields...)
		sort.Strings(sorted)
		if len(got) != len(sorted) {
			t.Errorf("%s 的 immutable 字段 = %v, want %v", entity, got, sorted)
			continue
		}
		for i := range got {
			if got[i] != sorted[i] {
				t.Errorf("%s 的 immutable 字段 = %v, want %v", entity, got, sorted)
				break
			}
		}
	}
	// 其余实体不得有 immutable 字段（菜单全部归运维）
	for _, entity := range []string{SeedEntityMenu, SeedEntityDepartment, SeedEntityPerson} {
		if fields := SeedImmutableFields(entity); len(fields) != 0 {
			t.Errorf("%s 不应有 immutable 字段，实际 %v", entity, fields)
		}
	}
}

// TestSeedImmutableFieldsDeclaredEntitiesExist 矩阵引用的实体标识必须是已声明的常量值，
// 防止把表名或拼错的实体名写进矩阵（会静默变成一个永远不生效的声明）。
func TestSeedImmutableFieldsDeclaredEntitiesExist(t *testing.T) {
	declared := map[string]bool{
		SeedEntityTenant:            true,
		SeedEntityDepartment:        true,
		SeedEntityApplication:       true,
		SeedEntityApplicationClient: true,
		SeedEntityMenu:              true,
		SeedEntityRole:              true,
		SeedEntityPerson:            true,
		SeedEntityUser:              true,
	}
	for _, authority := range SeedFieldAuthorities {
		if !declared[authority.Entity] {
			t.Errorf("矩阵使用了未声明的实体标识 %q（%s）", authority.Entity, authority.Field)
		}
	}
}
