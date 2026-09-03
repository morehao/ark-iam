package svctenant

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
)

// TestApiKeyOwnerModel 验证密钥归属模型：
// 个人密钥=真实用户本人（无需 super 自助创建/吊销）；服务账号密钥=开发者模式（需 super 创建/管理）。
func TestApiKeyOwnerModel(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApiKeyEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	tenantID := "t1"
	superOp := seedTestOperator(t, tenantID, true)
	self := seedTestOperator(t, tenantID, false)
	machine := &model.UserEntity{
		TenantID:   tenantID,
		UserType:   model.UserTypeMachine,
		Name:       "svc-notify",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}
	if err := dbclient.IamDB(context.Background()).Create(machine).Error; err != nil {
		t.Fatalf("seed machine owner: %v", err)
	}

	svc := NewApiKeySvc()

	// 个人密钥：成员自助创建，明文仅此一次返回
	selfResp, err := svc.Create(newTestTenantCtx(tenantID, self.ID), &dtotenant.ApiKeyCreateReq{Name: "my-key"})
	if err != nil {
		t.Fatalf("self create personal key: %v", err)
	}
	if selfResp.Key == "" || selfResp.KeyPrefix == "" {
		t.Fatal("expected one-time plaintext key and prefix")
	}

	// 默认列表 = 本人密钥（成员无权看服务账号/全量）
	myPage, err := svc.PageList(newTestTenantCtx(tenantID, self.ID), &dtotenant.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("self page list: %v", err)
	}
	if myPage.Total != 1 || myPage.List[0].OwnerUserID != self.ID {
		t.Fatalf("personal list mismatch: %+v", myPage)
	}

	// 成员试图为服务账号创建/查看密钥被拒（需系统管理能力）
	if _, err := svc.Create(newTestTenantCtx(tenantID, self.ID), &dtotenant.ApiKeyCreateReq{Name: "k", MachineUserID: machine.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member create machine key: want system admin required, got %v", err)
	}
	if _, err := svc.PageList(newTestTenantCtx(tenantID, self.ID), &dtotenant.ApiKeyPageListReq{MachineUserID: machine.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member list machine keys: want system admin required, got %v", err)
	}

	// super 为服务账号创建密钥（开发者模式），归属 machine
	machineResp, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyCreateReq{Name: "svc-key", MachineUserID: machine.ID})
	if err != nil {
		t.Fatalf("super create machine key: %v", err)
	}
	if machineResp.ID == "" {
		t.Fatal("expected machine key id")
	}
	machinePage, err := svc.PageList(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyPageListReq{MachineUserID: machine.ID})
	if err != nil {
		t.Fatalf("super list machine keys: %v", err)
	}
	if machinePage.Total != 1 ||
		machinePage.List[0].OwnerUserID != machine.ID ||
		machinePage.List[0].OwnerType != string(model.UserTypeMachine) ||
		machinePage.List[0].OwnerName != "svc-notify" {
		t.Fatalf("machine key list mismatch: %+v", machinePage)
	}

	// 成员不能吊销/删除他人(服务账号)密钥；本人密钥成员自管
	if err := svc.Revoke(newTestTenantCtx(tenantID, self.ID), &dtotenant.ApiKeyRevokeReq{ApiKeyID: machineResp.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member revoke machine key: want system admin required, got %v", err)
	}
	if err := svc.Revoke(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyRevokeReq{ApiKeyID: machineResp.ID}); err != nil {
		t.Fatalf("super revoke machine key: %v", err)
	}
	if err := svc.Revoke(newTestTenantCtx(tenantID, self.ID), &dtotenant.ApiKeyRevokeReq{ApiKeyID: selfResp.ID}); err != nil {
		t.Fatalf("self revoke personal key: %v", err)
	}
}

// TestApiKeyDeleteRequiresOwnerOrSuper 验证删除权限与删除后列表不再出现。
func TestApiKeyDeleteRequiresOwnerOrSuper(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApiKeyEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	tenantID := "t1"
	superOp := seedTestOperator(t, tenantID, true)
	alice := seedTestOperator(t, tenantID, false)
	bob := seedTestOperator(t, tenantID, false)

	svc := NewApiKeySvc()
	aliceKey, err := svc.Create(newTestTenantCtx(tenantID, alice.ID), &dtotenant.ApiKeyCreateReq{Name: "alice-key"})
	if err != nil {
		t.Fatalf("alice create: %v", err)
	}

	// bob 不能删除 alice 的密钥
	if err := svc.Delete(newTestTenantCtx(tenantID, bob.ID), &dtotenant.ApiKeyDeleteReq{ApiKeyID: aliceKey.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("bob delete alice key: want system admin required, got %v", err)
	}
	// super 可以删除任意个人密钥
	if err := svc.Delete(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyDeleteReq{ApiKeyID: aliceKey.ID}); err != nil {
		t.Fatalf("super delete alice key: %v", err)
	}
	page, err := svc.PageList(newTestTenantCtx(tenantID, alice.ID), &dtotenant.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("alice list after delete: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected 0 keys after delete, got %d", page.Total)
	}
}
