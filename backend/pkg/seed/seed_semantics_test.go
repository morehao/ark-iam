package seed_test

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// TestRunReportsChanges 变更报告（可观测性）：全新库首次播种必须报告各实体的 created；
// 二次执行必须零 created（幂等），部署时据此核对"这次启动改了什么"。
func TestRunReportsChanges(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	rep, err := seed.Run(ctx, db)
	if err != nil {
		t.Fatalf("seed run fail: %v", err)
	}
	created := map[string]int{}
	for _, change := range rep.Changes {
		if change.Action == "created" {
			created[change.Entity]++
		}
	}
	wantCreated := map[string]int{
		model.SeedEntityTenant:            1,
		model.SeedEntityDepartment:        1,
		model.SeedEntityApplication:       2,
		model.SeedEntityMenu:              15,
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

	rep2, err := seed.Run(ctx, db)
	if err != nil {
		t.Fatalf("seed (2nd) run fail: %v", err)
	}
	for _, change := range rep2.Changes {
		if change.Action == "created" {
			t.Errorf("二次执行不得再有 created: %+v", change)
		}
		if change.Action == "migrated" {
			t.Errorf("二次执行不得再触发迁移: %+v", change)
		}
	}
}

// TestSeedIamRespectsOperatorOwnedFields 单一写者语义：
//   - 运维字段（平台租户名、根部门名、应用启停/排序、客户端回调地址）改过之后，种子不得回写；
//   - 种子字段（应用/客户端名称、客户端 source）仍按矩阵收敛；
//   - 安全性不变式（平台租户 status=active）仍被纠正。
func TestSeedIamRespectsOperatorOwnedFields(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed fail: %v", err)
	}

	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query tenant: %v", err)
	}
	// 运维改动：平台租户名（迁移只在历史种子值上触发，这里已是运维自定义值）
	if err := db.Model(&model.TenantEntity{}).Where("id = ?", tenant.ID).
		Updates(map[string]any{"name": "ACME 平台运营中心", "status": model.TenantStatusSuspended}).Error; err != nil {
		t.Fatalf("degrade tenant: %v", err)
	}
	// 运维改动：根部门名按自己组织架构命名
	if err := db.Model(&model.DepartmentEntity{}).Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").
		Update("name", "ACME 技术中心").Error; err != nil {
		t.Fatalf("degrade department: %v", err)
	}
	// 运维改动：应用启停/排序（create_only）
	if err := db.Model(&model.ApplicationEntity{}).Where("code = ?", "platform_admin").
		Updates(map[string]any{"status": model.AppStatusDisable, "sort": 9}).Error; err != nil {
		t.Fatalf("degrade application: %v", err)
	}
	// 运维改动：客户端回调地址（环境相关，create_only）
	customRedirect := `["https://sso.example.com/auth/callback"]`
	if err := db.Model(&model.ApplicationClientEntity{}).Where("code = ?", "platform-admin-web").
		Update("redirect_uris", []byte(customRedirect)).Error; err != nil {
		t.Fatalf("degrade client: %v", err)
	}

	if err := seed.SeedIam(ctx, db); err != nil {
		t.Fatalf("seed (2nd) fail: %v", err)
	}

	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query tenant after reseed: %v", err)
	}
	if tenant.Name != "ACME 平台运营中心" {
		t.Errorf("平台租户名 = %q, want 运维自定义值（migrate_once 不得覆盖运维改名）", tenant.Name)
	}
	if tenant.Status != model.TenantStatusActive {
		t.Errorf("平台租户 status = %q, want %q（reconcile 不变式）", tenant.Status, model.TenantStatusActive)
	}

	var rootDept model.DepartmentEntity
	if err := db.Where("tenant_id = ? AND parent_id = ?", tenant.ID, "").First(&rootDept).Error; err != nil {
		t.Fatalf("query root department: %v", err)
	}
	if rootDept.Name != "ACME 技术中心" {
		t.Errorf("根部门名 = %q, want 运维自定义值（派生字段只在历史种子值上同步一次）", rootDept.Name)
	}

	var adminApp model.ApplicationEntity
	if err := db.Where("code = ?", "platform_admin").First(&adminApp).Error; err != nil {
		t.Fatalf("query application: %v", err)
	}
	if adminApp.Status != model.AppStatusDisable || adminApp.Sort != 9 {
		t.Errorf("应用 status/sort = (%q, %d), want (disable, 9)：create_only 字段种子不得回写", adminApp.Status, adminApp.Sort)
	}
	if adminApp.Name != "平台管理后台" {
		t.Errorf("应用名 = %q, want 平台管理后台（reconcile 字段必须收敛）", adminApp.Name)
	}

	var client model.ApplicationClientEntity
	if err := db.Where("code = ?", "platform-admin-web").First(&client).Error; err != nil {
		t.Fatalf("query application_client: %v", err)
	}
	if string(client.RedirectURIs) != customRedirect {
		t.Errorf("客户端回调地址 = %s, want %s（create_only 字段种子不得回写）", client.RedirectURIs, customRedirect)
	}
	if client.Name != "平台管理后台" {
		t.Errorf("客户端名 = %q, want 平台管理后台（reconcile 字段必须收敛）", client.Name)
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
				Config: []byte(`{}`), GrantedScope: []byte(`[]`),
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
