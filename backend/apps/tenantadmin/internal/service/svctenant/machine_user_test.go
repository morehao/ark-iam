package svctenant

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

// seedTestOperator 播种租户 tenantID 下的真实操作者；super=true 时授予内置超级角色。
func seedTestOperator(t *testing.T, tenantID string, super bool) *model.UserEntity {
	t.Helper()
	db := dbclient.IamDB(context.Background())
	op := &model.UserEntity{
		TenantID:   tenantID,
		UserType:   model.UserTypeMember,
		Name:       "operator",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}
	if err := db.Create(op).Error; err != nil {
		t.Fatalf("seed operator: %v", err)
	}
	if !super {
		return op
	}
	role := &model.RoleEntity{
		TenantID:   tenantID,
		AppID:      "app-admin",
		Code:       "test-admin",
		Name:       "测试超级角色",
		Source:     string(model.RoleSourceBuiltin),
		AdminLevel: string(model.SysAdminLevelSuper),
	}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed super role: %v", err)
	}
	ur := &model.UserRoleEntity{TenantID: tenantID, UserID: op.ID, RoleID: role.ID}
	if err := db.Create(ur).Error; err != nil {
		t.Fatalf("bind super role: %v", err)
	}
	return op
}

func newTestTenantCtx(tenantID, userID string) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

// seedTestOrg 播种租户下的组织节点。
func seedTestOrg(t *testing.T, tenantID, name string) *model.OrganizationEntity {
	t.Helper()
	db := dbclient.IamDB(context.Background())
	org := &model.OrganizationEntity{
		TenantID: tenantID,
		ParentID: "",
		Name:     name,
		Sort:     0,
		Status:   string(model.OrgNodeStatusActive),
	}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("seed org %s: %v", name, err)
	}
	return org
}

// TestMachineUserOrgLifecycleAndGuards 覆盖服务账号组织归属生命周期与守卫：
// 创建(需 super+主部门) → 列表主部门 → 详情归属 → 改主部门/清参与 → 挂起 → 角色(禁授 super) →
// 删除(有 key 拒绝;成功后级联清理角色与部门关系)。
func TestMachineUserOrgLifecycleAndGuards(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{},
		&model.ApplicationEntity{}, &model.ApiKeyEntity{}, &model.PersonEntity{},
		&model.OrganizationEntity{}, &model.OrganizationUserEntity{})
	tenantID := "t1"
	superOp := seedTestOperator(t, tenantID, true)
	memberOp := seedTestOperator(t, tenantID, false)
	rd := seedTestOrg(t, tenantID, "研发部")
	op := seedTestOrg(t, tenantID, "运维部")
	fin := seedTestOrg(t, tenantID, "财务部")

	svc := NewMachineUserSvc()

	// 非 super 创建被拒
	_, err := svc.Create(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.MachineUserCreateReq{
		Name:            "forbidden",
		OrganizationIDs: []string{rd.ID},
	})
	if err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member create: want system admin required, got %v", err)
	}
	// 缺主部门 / 多主部门被拒
	if _, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserCreateReq{Name: "no-org"}); err != code.GetError(code.MachineUserOrgRequiredError) {
		t.Fatalf("create without primary org: want org required, got %v", err)
	}
	if _, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserCreateReq{
		Name: "two-primary", OrganizationIDs: []string{rd.ID, op.ID},
	}); err != code.GetError(code.MachineUserOrgRequiredError) {
		t.Fatalf("create with two primary orgs: want org required, got %v", err)
	}
	// 跨租户组织被拒
	if _, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserCreateReq{
		Name: "cross", OrganizationIDs: []string{"org-other-tenant"},
	}); err != code.GetError(code.OrganizationNotExistError) {
		t.Fatalf("create with foreign org: want not exist, got %v", err)
	}

	// super 创建：主部门 rd + 参与 op
	created, err := svc.Create(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserCreateReq{
		Name: "svc-pay", Description: "支付回调", OrganizationIDs: []string{rd.ID}, SecondaryOrgIDs: []string{op.ID},
	})
	if err != nil {
		t.Fatalf("create machine user: %v", err)
	}
	machineID := created.MachineUserID

	// 组织归属落库：1 primary(rd) + 1 secondary(op)
	relList, err := dao.NewOrganizationUserDao().GetListByCond(newTestTenantCtx(tenantID, superOp.ID), &dao.OrganizationUserCond{TenantID: tenantID, UserID: machineID})
	if err != nil {
		t.Fatalf("query org relations: %v", err)
	}
	byType := map[model.OrgUserRelationType]string{}
	for _, r := range relList {
		byType[r.RelationType] = r.OrganizationID
	}
	if byType[model.OrgUserRelationPrimary] != rd.ID || byType[model.OrgUserRelationSecondary] != op.ID {
		t.Fatalf("org relations mismatch: %+v", byType)
	}

	// 列表：主部门名称回填
	page, err := svc.PageList(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserPageListReq{Name: "svc-pay"})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if page.Total != 1 || page.List[0].MachineUserID != machineID ||
		page.List[0].PrimaryOrgID != rd.ID || page.List[0].PrimaryOrgName != "研发部" {
		t.Fatalf("page list mismatch: %+v", page)
	}

	// 详情：组织归属 + 角色
	detail, err := svc.Detail(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserDetailReq{MachineUserID: machineID})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Name != "svc-pay" || len(detail.Roles) != 0 {
		t.Fatalf("detail mismatch: %+v", detail)
	}
	foundPrimary, foundSecondary := false, false
	for _, org := range detail.Organizations {
		switch org.RelationType {
		case model.OrgUserRelationPrimary:
			foundPrimary = org.OrganizationID == rd.ID
		case model.OrgUserRelationSecondary:
			foundSecondary = org.OrganizationID == op.ID
		}
	}
	if !foundPrimary || !foundSecondary {
		t.Fatalf("detail organizations mismatch: %+v", detail.Organizations)
	}

	// 更新：改主部门→fin、参与部门全量替换为 [op, fin]
	newSecondary := []string{op.ID, fin.ID}
	if err := svc.Update(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserUpdateReq{
		MachineUserID: machineID, Name: "svc-pay-v2", Description: "支付回调V2",
		PrimaryOrgID: &fin.ID, SecondaryOrgIDs: &newSecondary,
	}); err != nil {
		t.Fatalf("update org: %v", err)
	}
	relList, err = dao.NewOrganizationUserDao().GetListByCond(newTestTenantCtx(tenantID, superOp.ID), &dao.OrganizationUserCond{TenantID: tenantID, UserID: machineID})
	if err != nil {
		t.Fatalf("re-query org relations: %v", err)
	}
	byType = map[model.OrgUserRelationType]string{}
	for _, r := range relList {
		byType[r.RelationType] = r.OrganizationID
	}
	if byType[model.OrgUserRelationPrimary] != fin.ID {
		t.Fatalf("primary should move to fin, got %s", byType[model.OrgUserRelationPrimary])
	}

	// 主部门不可清空
	if err := svc.Update(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserUpdateReq{
		MachineUserID: machineID, Name: "svc-pay-v2", PrimaryOrgID: strPtr(""),
	}); err != code.GetError(code.MachineUserOrgRequiredError) {
		t.Fatalf("clear primary: want org required, got %v", err)
	}
	// 挂起
	if err := svc.UpdateStatus(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserStatusReq{MachineUserID: machineID, IsSuspended: true}); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	reloaded, err := dao.NewUserDao().GetByID(newTestTenantCtx(tenantID, superOp.ID), machineID)
	if err != nil || reloaded == nil {
		t.Fatalf("reload machine user: %v", err)
	}
	if reloaded.Name != "svc-pay-v2" || !reloaded.IsSuspended {
		t.Fatalf("reload mismatch: %+v", reloaded)
	}

	// 普通角色可授、super 角色禁授（按应用授权：role.app_id 须与 req.AppID 一致）
	devRole := &model.RoleEntity{
		TenantID: tenantID, AppID: "app-tenant", Code: "dev", Name: "开发者",
		Source: string(model.RoleSourceCustom), AdminLevel: string(model.SysAdminLevelMember),
	}
	if err := db.Create(devRole).Error; err != nil {
		t.Fatalf("seed dev role: %v", err)
	}
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-tenant", RoleIDs: []string{devRole.ID}}); err != nil {
		t.Fatalf("grant dev role: %v", err)
	}
	superRole := &model.RoleEntity{
		TenantID: tenantID, AppID: "app-tenant", Code: "tenant-owner", Name: "租户管理员",
		Source: string(model.RoleSourceBuiltin), AdminLevel: string(model.SysAdminLevelSuper),
	}
	if err := db.Create(superRole).Error; err != nil {
		t.Fatalf("seed super role 2: %v", err)
	}
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-tenant", RoleIDs: []string{superRole.ID}}); err != code.GetError(code.UserSuperRoleAssignForbidden) {
		t.Fatalf("grant super role to machine: want forbidden, got %v", err)
	}

	// 按应用隔离：授另一个应用的普通角色后，原应用角色不受影响；跨应用角色拒绝
	opsRole := &model.RoleEntity{
		TenantID: tenantID, AppID: "app-other", Code: "ops", Name: "运维",
		Source: string(model.RoleSourceCustom), AdminLevel: string(model.SysAdminLevelMember),
	}
	if err := db.Create(opsRole).Error; err != nil {
		t.Fatalf("seed ops role: %v", err)
	}
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-other", RoleIDs: []string{opsRole.ID}}); err != nil {
		t.Fatalf("grant ops role: %v", err)
	}
	var machRoleIDs []string
	if err := db.Model(&model.UserRoleEntity{}).Where("tenant_id = ? AND user_id = ?", tenantID, machineID).Pluck("role_id", &machRoleIDs).Error; err != nil {
		t.Fatalf("query machine roles: %v", err)
	}
	hasRole := func(ids []string, id string) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}
	if len(machRoleIDs) != 2 || !hasRole(machRoleIDs, devRole.ID) || !hasRole(machRoleIDs, opsRole.ID) {
		t.Fatalf("expected dev+ops roles kept per-app, got %+v", machRoleIDs)
	}
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-tenant", RoleIDs: []string{opsRole.ID}}); err != code.GetError(code.RoleNotExistError) {
		t.Fatalf("cross-app role grant: want RoleNotExist, got %v", err)
	}

	// 服务账号下有 key 时禁止删除
	key := &model.ApiKeyEntity{
		TenantID: tenantID, OwnerUserID: machineID, Name: "machine-key",
		KeyHash: "h", KeyPrefix: "prefix", Scope: json.RawMessage(`{}`), CreatedBy: superOp.ID,
	}
	if err := db.Create(key).Error; err != nil {
		t.Fatalf("seed machine key: %v", err)
	}
	if err := svc.Delete(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserDeleteReq{MachineUserID: machineID}); err != code.GetError(code.MachineUserDeleteHasKeysError) {
		t.Fatalf("delete with keys: want has-keys forbidden, got %v", err)
	}
	// 清掉 key 后删除成功：用户软删 + 角色/部门关系级联清理
	if err := dao.NewApiKeyDao().Delete(newTestTenantCtx(tenantID, superOp.ID), key.ID, superOp.ID); err != nil {
		t.Fatalf("delete key: %v", err)
	}
	if err := svc.Delete(newTestTenantCtx(tenantID, superOp.ID), &dtotenant.MachineUserDeleteReq{MachineUserID: machineID}); err != nil {
		t.Fatalf("delete machine user: %v", err)
	}
	gone, err := dao.NewUserDao().GetByID(newTestTenantCtx(tenantID, superOp.ID), machineID)
	if err != nil {
		t.Fatalf("query deleted machine user: %v", err)
	}
	if gone != nil {
		t.Fatal("machine user should be soft-deleted")
	}
	var orgRelCount int64
	if err := db.Model(&model.OrganizationUserEntity{}).Where("user_id = ?", machineID).Count(&orgRelCount).Error; err != nil {
		t.Fatalf("count org relations: %v", err)
	}
	if orgRelCount != 0 {
		t.Fatalf("org relations should be cascade cleaned, got %d", orgRelCount)
	}
	var roleRelCount int64
	if err := db.Model(&model.UserRoleEntity{}).Where("user_id = ?", machineID).Count(&roleRelCount).Error; err != nil {
		t.Fatalf("count role relations: %v", err)
	}
	if roleRelCount != 0 {
		t.Fatalf("role relations should be cascade cleaned, got %d", roleRelCount)
	}
}

func strPtr(s string) *string {
	return &s
}

// TestServiceAccountCannotBeLeader 服务账号不可被设为主管(leader)；参与(secondary)关系允许。
func TestServiceAccountCannotBeLeader(t *testing.T) {
	testutil.SetupSQLite(t, &model.UserEntity{}, &model.OrganizationEntity{}, &model.OrganizationUserEntity{},
		&model.RoleEntity{}, &model.UserRoleEntity{})
	tenantID := "t1"
	superOp := seedTestOperator(t, tenantID, true)
	org := seedTestOrg(t, tenantID, "研发部")
	machine := &model.UserEntity{
		TenantID:   tenantID,
		UserType:   model.UserTypeMachine,
		Name:       "svc-pay",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
	}
	if err := dbclient.IamDB(context.Background()).Create(machine).Error; err != nil {
		t.Fatalf("seed machine: %v", err)
	}

	svc := NewOrganizationUserSvc()
	ctx := newTestTenantCtx(tenantID, superOp.ID)
	// leader：拒绝
	if _, err := svc.Create(ctx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: org.ID,
		UserID:         machine.ID,
		RelationType:   model.OrgUserRelationLeader,
	}); err != code.GetError(code.UserMemberOperationOnlyError) {
		t.Fatalf("machine as leader: want member-operation-only, got %v", err)
	}
	// secondary：允许
	if _, err := svc.Create(ctx, &dtotenant.OrganizationUserCreateReq{
		OrganizationID: org.ID,
		UserID:         machine.ID,
		RelationType:   model.OrgUserRelationSecondary,
	}); err != nil {
		t.Fatalf("machine as secondary member: %v", err)
	}
	// 已挂 primary 关系的服务账号收敛为 leader 同样拒绝
	if err := svc.Update(ctx, &dtotenant.OrganizationUserUpdateReq{
		OrganizationID: org.ID,
		UserID:         machine.ID,
		RelationType:   model.OrgUserRelationLeader,
	}); err != code.GetError(code.UserMemberOperationOnlyError) {
		t.Fatalf("converge machine to leader: want member-operation-only, got %v", err)
	}
}
