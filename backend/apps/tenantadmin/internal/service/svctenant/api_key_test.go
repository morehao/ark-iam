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

// TestApiKeyServiceAccountOnly 验证 API 密钥已收敛为「仅服务账号」管理模型：
// 创建/列表/吊销/删除均为系统管理操作；普通成员无任何密钥通道（个人密钥已下线）。
func TestApiKeyServiceAccountOnly(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApiKeyEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	tenantID := "t1"
	superOp := seedTestOperator(t, tenantID, true)
	memberOp := seedTestOperator(t, tenantID, false)
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
	machine2 := &model.UserEntity{
		TenantID:   tenantID,
		UserType:   model.UserTypeMachine,
		Name:       "svc-webhook",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}
	if err := dbclient.IamDB(context.Background()).Create(machine2).Error; err != nil {
		t.Fatalf("seed machine owner 2: %v", err)
	}

	svc := NewApiKeySvc()

	// 普通成员无任何密钥操作通道
	if _, err := svc.Create(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.ApiKeyCreateReq{Name: "k", MachineUserID: machine.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member create: want system admin required, got %v", err)
	}
	if _, err := svc.PageList(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.ApiKeyPageListReq{}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member page list: want system admin required, got %v", err)
	}

	// super 为服务账号创建密钥：明文仅一次返回、归属 machine
	created, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyCreateReq{Name: "svc-key", MachineUserID: machine.ID})
	if err != nil {
		t.Fatalf("super create: %v", err)
	}
	if created.Key == "" || created.KeyPrefix == "" {
		t.Fatal("expected one-time plaintext key and prefix")
	}

	// 成员对已存在的密钥也无吊销/删除通道
	if err := svc.Revoke(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.ApiKeyRevokeReq{ApiKeyID: created.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member revoke: want system admin required, got %v", err)
	}
	if err := svc.Delete(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.ApiKeyDeleteReq{ApiKeyID: created.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member delete: want system admin required, got %v", err)
	}

	// 指定服务账号过滤列表
	machinePage, err := svc.PageList(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyPageListReq{MachineUserID: machine.ID})
	if err != nil {
		t.Fatalf("super list by machine: %v", err)
	}
	if machinePage.Total != 1 ||
		machinePage.List[0].OwnerUserID != machine.ID ||
		machinePage.List[0].OwnerType != string(model.UserTypeMachine) ||
		machinePage.List[0].OwnerName != "svc-notify" {
		t.Fatalf("machine key list mismatch: %+v", machinePage)
	}

	// 租户全部密钥（不带过滤）同样只见服务账号密钥
	allPage, err := svc.PageList(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("super list all: %v", err)
	}
	if allPage.Total != 1 {
		t.Fatalf("expected 1 key total, got %d", allPage.Total)
	}

	// 第二个服务账号的密钥互不可见（过滤正确）
	created2, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyCreateReq{Name: "webhook-key", MachineUserID: machine2.ID})
	if err != nil {
		t.Fatalf("super create key2: %v", err)
	}
	filterPage, err := svc.PageList(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyPageListReq{MachineUserID: machine2.ID})
	if err != nil {
		t.Fatalf("super list machine2: %v", err)
	}
	if filterPage.Total != 1 || filterPage.List[0].KeyID != created2.ID {
		t.Fatalf("machine2 filter mismatch: %+v", filterPage)
	}

	// 吊销后列表状态更新；再删除后消失
	if err := svc.Revoke(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyRevokeReq{ApiKeyID: created2.ID}); err != nil {
		t.Fatalf("super revoke: %v", err)
	}
	if err := svc.Delete(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyDeleteReq{ApiKeyID: created2.ID}); err != nil {
		t.Fatalf("super delete: %v", err)
	}
	afterPage, err := svc.PageList(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if afterPage.Total != 1 {
		t.Fatalf("expected 1 key after delete, got %d", afterPage.Total)
	}

	// 非服务账号主体不可作为归属（把真实用户当服务账号归属 → 不存在）
	if _, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyCreateReq{Name: "bad", MachineUserID: memberOp.ID}); err != code.GetError(code.ApiKeyOwnerNotExistError) {
		t.Fatalf("create with real-user owner: want owner not exist, got %v", err)
	}
}

// TestApiKeyDeleteRequiresSuper 验证密钥删除仅限系统管理能力（服务账号不可自登录，无"本人自管"通道）。
func TestApiKeyDeleteRequiresSuper(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApiKeyEntity{}, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	tenantID := "t1"
	superOp := seedTestOperator(t, tenantID, true)
	memberOp := seedTestOperator(t, tenantID, false)
	machine := &model.UserEntity{
		TenantID:   tenantID,
		UserType:   model.UserTypeMachine,
		Name:       "svc-alice",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}
	if err := dbclient.IamDB(context.Background()).Create(machine).Error; err != nil {
		t.Fatalf("seed machine owner: %v", err)
	}
	// 直接落一条既有密钥行，模拟历史数据
	key := &model.ApiKeyEntity{
		TenantID: tenantID, OwnerUserID: machine.ID, Name: "legacy",
		KeyHash: "h", KeyPrefix: "ak_", Scope: json.RawMessage(`{}`), CreatedBy: superOp.ID,
	}
	if err := dbclient.IamDB(context.Background()).Create(key).Error; err != nil {
		t.Fatalf("seed key: %v", err)
	}

	svc := NewApiKeySvc()
	if err := svc.Revoke(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.ApiKeyRevokeReq{ApiKeyID: key.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member revoke: want system admin required, got %v", err)
	}
	if err := svc.Delete(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.ApiKeyDeleteReq{ApiKeyID: key.ID}); err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member delete: want system admin required, got %v", err)
	}
	// 密钥仍在
	page, err := svc.PageList(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyPageListReq{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected 1 key, got %d", page.Total)
	}
	// super 吊销成功
	if err := svc.Revoke(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.ApiKeyRevokeReq{ApiKeyID: key.ID}); err != nil {
		t.Fatalf("super revoke: %v", err)
	}
}
