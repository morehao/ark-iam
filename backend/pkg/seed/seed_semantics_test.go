package seed_test

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// TestBootstrapReportsChanges 变更报告（可观测性）：L1 首次引导必须报告各实体的 created；
// 二次引导必须自锁并零写入（部署时据此核对"这次到底写了什么"）。
//
// 报告是初始化接口回给初始化页面的内容（见 install 服务），因此它必须如实反映本次真实写入——
// L1 只创建，故 Change.Action 恒为 created，不存在 updated/migrated。
func TestBootstrapReportsChanges(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	rep, status, err := seed.Bootstrap(ctx, db, testDefinition(t))
	if err != nil {
		t.Fatalf("bootstrap fail: %v", err)
	}
	if status != seed.StatusCreated {
		t.Fatalf("status = %q, want %q", status, seed.StatusCreated)
	}
	if rep.TenantID == "" {
		t.Error("报告必须带上平台租户 ID（审计与响应据此定位目标租户）")
	}
	created := map[string]int{}
	for _, change := range rep.Changes {
		if change.Action != "created" {
			t.Errorf("L1 只创建，动作必须恒为 created，实际 %+v", change)
		}
		created[change.Entity]++
	}
	wantCreated := map[string]int{
		model.SeedEntityTenant:            1,
		model.SeedEntityDepartment:        1,
		model.SeedEntityApplication:       2,
		model.SeedEntityMenu:              14,
		model.SeedEntityApplicationClient: 2,
		model.SeedEntityRole:              1,
		model.SeedEntityPerson:            1,
		model.SeedEntityUser:              1,
		model.TableNameTenantApplication:  1,
		model.TableNameDepartmentUser:     1,
		model.TableNameUserRole:           1,
	}
	for entity, want := range wantCreated {
		if created[entity] != want {
			t.Errorf("created[%s] = %d, want %d", entity, created[entity], want)
		}
	}
	if rep.Summary() == "" {
		t.Error("报告汇总不得为空")
	}

	rep2, status2, err := seed.Bootstrap(ctx, db, testDefinition(t))
	if err != nil {
		t.Fatalf("bootstrap (2nd) fail: %v", err)
	}
	if status2 != seed.StatusAlreadyInitialized {
		t.Errorf("二次引导 status = %q, want %q", status2, seed.StatusAlreadyInitialized)
	}
	if len(rep2.Changes) != 0 {
		t.Errorf("二次引导必须零写入，实际 %+v", rep2.Changes)
	}
}

// TestSeedAssociationPartialUniqueIndexes 关联表的并发幂等由唯一索引兜底（P4）：
// 活跃行重复插入必须失败；索引是"部分唯一"（deleted_at IS NULL），因为三张表都是软删，
// 全量唯一索引会让"软删后重新授权"撞上历史行。
func TestSeedAssociationPartialUniqueIndexes(t *testing.T) {
	db := setupDB(t)

	cases := []struct {
		name   string
		create func() any
	}{
		{"tenant_application", func() any {
			return &model.TenantApplicationEntity{
				TenantID: "t1", AppID: "a1", Status: model.TenantApplicationStatusEnable,
			}
		}},
		{"role_menu", func() any {
			return &model.RoleMenuEntity{TenantID: "t1", RoleID: "r1", MenuID: "m1"}
		}},
		{"user_role", func() any {
			return &model.UserRoleEntity{TenantID: "t1", UserID: "u1", RoleID: "r1"}
		}},
	}

	for _, tc := range cases {
		first := tc.create()
		if err := db.Create(first).Error; err != nil {
			t.Fatalf("%s: 首次插入失败: %v", tc.name, err)
		}
		if err := db.Create(tc.create()).Error; err == nil {
			t.Errorf("%s: 活跃行重复插入必须被唯一索引拒绝", tc.name)
		}
		if err := db.Delete(first).Error; err != nil {
			t.Fatalf("%s: 软删失败: %v", tc.name, err)
		}
		if err := db.Create(tc.create()).Error; err != nil {
			t.Errorf("%s: 软删后重新授权不得被索引拦住: %v", tc.name, err)
		}
	}
}
