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
// 准入判据是「被改会导致种子定位失效或鉴权被绕过」——展示字段不算，定位键与安全不变式才算。
func TestSeedOwnsFieldOnlyForReconcile(t *testing.T) {
	if !SeedOwnsField(SeedEntityApplication, "source") {
		t.Error("application.source 是安全不变式，应为种子收敛字段")
	}
	if SeedOwnsField(SeedEntityApplication, "name") {
		t.Error("application.name 属展示字段（create_only，归运维），不应算种子拥有")
	}
	if SeedOwnsField(SeedEntityApplication, "status") {
		t.Error("application.status 属 create_only，不应算种子拥有")
	}
	if SeedOwnsField(SeedEntityApplication, "undeclared_field") {
		t.Error("未声明字段应默认为 create_only（归运维）")
	}
	// 菜单：**没有任何 reconcile 字段**——种子改按 seed_key（种子身份键，非控制台字段）认行，
	// 于是 code/app_id 连同展示与结构字段一律归运维可改。
	for _, field := range []string{"seed_key", "app_id", "code", "name", "path", "icon", "sort", "component", "type", "visibility", "parent_id", "status"} {
		if SeedOwnsField(SeedEntityMenu, field) {
			t.Errorf("menu.%s 归运维（create_only），不应算种子拥有", field)
		}
	}
	if SeedOwnsField(SeedEntityApplicationClient, "name") {
		t.Error("application_client.name 属展示字段（create_only，归运维），不应算种子拥有")
	}
	if got := SeedFieldModeOf(SeedEntityTenant, "name"); got != SeedFieldMigrateOnce {
		t.Errorf("tenant.name mode = %q, want %q（改名必须走一次性迁移，不能无条件收敛）", got, SeedFieldMigrateOnce)
	}
	if got := SeedFieldModeOf(SeedEntityDepartment, "name"); got != SeedFieldMigrateOnce {
		t.Errorf("department.name mode = %q, want %q", got, SeedFieldMigrateOnce)
	}
}

// TestSeedReconcileFields 收敛字段清单必须与矩阵一致（种子据此生成更新集）：
// 菜单**没有任何收敛字段**——种子只按 seed_key 认行 + 缺失时创建，字段全部归运维。
func TestSeedReconcileFields(t *testing.T) {
	fields := SeedReconcileFields(SeedEntityMenu)
	if len(fields) != 0 {
		t.Errorf("menu 收敛字段 = %v，期望为空（全部字段归运维，定位由 seed_key 承担）", fields)
	}
	// 应用仍保留安全不变式：source 是唯一收敛字段
	appFields := map[string]bool{}
	for _, field := range SeedReconcileFields(SeedEntityApplication) {
		appFields[field] = true
	}
	if !appFields["source"] {
		t.Error("application 收敛字段应包含 source（内置标记，安全不变式）")
	}
	for _, field := range []string{"seed_key", "code", "name", "description", "sort", "status"} {
		if appFields[field] {
			t.Errorf("application.%s 归运维（create_only），不应出现在收敛集里", field)
		}
	}
}
