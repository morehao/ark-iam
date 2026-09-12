package model

import "testing"

// TestSeedAuthorityMatrixIsUniqueAndWellFormed 矩阵自身的不变式：
// (entity, field) 唯一、mode 合法、键非空——矩阵是种子与控制台的共同真相源，声明错误会静默改变写者。
func TestSeedAuthorityMatrixIsUniqueAndWellFormed(t *testing.T) {
	allowed := map[SeedFieldMode]bool{
		SeedFieldReconcile:   true,
		SeedFieldCreateOnly:  true,
		SeedFieldMigrateOnce: true,
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

// TestSeedOwnsFieldOnlyForReconcile 只有 reconcile 字段算"种子拥有"（控制台须拒写）：
// create_only 归运维、migrate_once 由迁移清单单独执行、未声明字段默认归运维。
func TestSeedOwnsFieldOnlyForReconcile(t *testing.T) {
	if !SeedOwnsField(SeedEntityApplication, "name") {
		t.Error("application.name 应为种子收敛字段")
	}
	if SeedOwnsField(SeedEntityApplication, "status") {
		t.Error("application.status 属 create_only，不应算种子拥有")
	}
	if SeedOwnsField(SeedEntityApplication, "undeclared_field") {
		t.Error("未声明字段应默认为 create_only（归运维）")
	}
	if got := SeedFieldModeOf(SeedEntityTenant, "name"); got != SeedFieldMigrateOnce {
		t.Errorf("tenant.name mode = %q, want %q（改名必须走一次性迁移，不能无条件收敛）", got, SeedFieldMigrateOnce)
	}
	if got := SeedFieldModeOf(SeedEntityDepartment, "name"); got != SeedFieldMigrateOnce {
		t.Errorf("department.name mode = %q, want %q", got, SeedFieldMigrateOnce)
	}
}

// TestSeedReconcileFields 收敛字段清单必须与矩阵一致（种子据此生成更新集）。
func TestSeedReconcileFields(t *testing.T) {
	got := map[string]bool{}
	for _, field := range SeedReconcileFields(SeedEntityMenu) {
		got[field] = true
	}
	for _, field := range []string{"code", "parent_id", "name", "path", "icon", "sort", "component", "type", "visibility"} {
		if !got[field] {
			t.Errorf("menu 收敛字段缺少 %q", field)
		}
	}
	if got["status"] {
		t.Error("menu.status 属 create_only（运维可停用菜单），不应出现在收敛集里")
	}
}
