package svctenant

import (
	"testing"

	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/apps/tenantadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/gorm"
)

// setupInviteTestDB 建库（邀请 + 角色/成员关系，用于管理能力判定）。
func setupInviteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.SetupSQLite(t,
		&model.InviteEntity{},
		&model.RoleEntity{},
		&model.UserRoleEntity{},
	)
}

// TestInviteWriteRequiresSystemAdmin 回归 P1-5：邀请生成/撤销/列表都是管理操作，
// 普通成员（无管理员角色）调用必须被拒（UserSystemAdminRequiredError），
// 否则任意成员可把外部人员拉进租户或读取有效邀请码。
func TestInviteWriteRequiresSystemAdmin(t *testing.T) {
	db := setupInviteTestDB(t)
	svc := NewInviteSvc(nil)

	normalCtx := newTestTenantCtx("t1", "op-normal")

	if _, err := svc.Create(requestCtx(normalCtx), &dtotenant.InviteCreateReq{
		Operator: operatorOf(normalCtx),
	}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("non-admin invite create: want UserSystemAdminRequiredError, got %v", err)
	}

	if _, err := svc.PageList(requestCtx(normalCtx), &dtotenant.InvitePageListReq{
		Operator: operatorOf(normalCtx),
	}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("non-admin invite list: want UserSystemAdminRequiredError, got %v", err)
	}

	// 播种一条邀请，非管理员撤销也必须被拒。
	adminCtx := newTestTenantCtx("t1", "op-admin")
	seedTestAdminOperator(t, db, "t1", "op-admin")
	resp, err := svc.Create(requestCtx(adminCtx), &dtotenant.InviteCreateReq{
		Operator: operatorOf(adminCtx),
	})
	if err != nil {
		t.Fatalf("admin invite create should succeed: %v", err)
	}

	if err := svc.Revoke(requestCtx(normalCtx), &dtotenant.InviteRevokeReq{
		InviteID: resp.InviteID, Operator: operatorOf(normalCtx),
	}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("non-admin invite revoke: want UserSystemAdminRequiredError, got %v", err)
	}

	// 管理员撤销应成功。
	if err := svc.Revoke(requestCtx(adminCtx), &dtotenant.InviteRevokeReq{
		InviteID: resp.InviteID, Operator: operatorOf(adminCtx),
	}); err != nil {
		t.Fatalf("admin invite revoke should succeed: %v", err)
	}
}
