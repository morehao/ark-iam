package svctenant

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
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
		TenantID:  tenantID,
		AppID:     "app-admin",
		Name:      "测试超级角色",
		Source:    model.RoleSourceBuiltin,
		AdminType: model.SysAdminTypeAdmin,
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

// seedTestDept 播种租户下的部门节点。
func seedTestDept(t *testing.T, tenantID, name string) *model.DepartmentEntity {
	t.Helper()
	db := dbclient.IamDB(context.Background())
	dept := &model.DepartmentEntity{
		TenantID: tenantID,
		ParentID: "",
		Name:     name,
		Sort:     0,
		Status:   model.DeptNodeStatusEnable,
	}
	if err := db.Create(dept).Error; err != nil {
		t.Fatalf("seed dept %s: %v", name, err)
	}
	return dept
}

// TestMachineUserDeptLifecycleAndGuards 覆盖服务账号部门归属生命周期与守卫：
// 创建(需 super+主部门) → 列表主部门 → 详情归属 → 改主部门/清参与 → 挂起 → 角色(禁授 super) →
// 删除(有 key 拒绝;成功后级联清理角色与部门关系)。
func TestMachineUserDeptLifecycleAndGuards(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{},
		&model.ApplicationEntity{}, &model.ApiKeyEntity{}, &model.PersonEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{})
	tenantID := "t1"
	adminOp := seedTestOperator(t, tenantID, true)
	memberOp := seedTestOperator(t, tenantID, false)
	rd := seedTestDept(t, tenantID, "研发部")
	op := seedTestDept(t, tenantID, "运维部")
	fin := seedTestDept(t, tenantID, "财务部")

	svc := NewMachineUserSvc()

	// 非 super 创建被拒
	_, err := svc.Create(newTestTenantCtx(tenantID, memberOp.ID), &dtotenant.MachineUserCreateReq{
		Name:                "forbidden",
		PrimaryDepartmentID: rd.ID,
	})
	if err != code.GetError(code.UserSystemAdminRequiredError) {
		t.Fatalf("member create: want system admin required, got %v", err)
	}
	// 缺主部门被拒（主部门为单值入参，"多主部门"在类型层面已不可表达）
	if _, err := svc.Create(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserCreateReq{Name: "no-dept"}); err != code.GetError(code.MachineUserDepartmentRequiredError) {
		t.Fatalf("create without primary dept: want dept required, got %v", err)
	}
	// 跨租户部门被拒
	if _, err := svc.Create(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserCreateReq{
		Name: "cross", PrimaryDepartmentID: "dept-other-tenant",
	}); err != code.GetError(code.DepartmentNotExistError) {
		t.Fatalf("create with foreign dept: want not exist, got %v", err)
	}

	// super 创建：主部门 rd + 参与 op
	created, err := svc.Create(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserCreateReq{
		Name: "svc-pay", Description: "支付回调", PrimaryDepartmentID: rd.ID, SecondaryDepartmentIDs: []string{op.ID},
	})
	if err != nil {
		t.Fatalf("create machine user: %v", err)
	}
	machineID := created.MachineUserID

	// 部门归属落库：1 primary(rd) + 1 secondary(op)
	relList, err := dao.NewDepartmentUserDao().GetListByCond(newTestTenantCtx(tenantID, adminOp.ID), &dao.DepartmentUserCond{TenantID: tenantID, UserID: machineID})
	if err != nil {
		t.Fatalf("query dept relations: %v", err)
	}
	byType := map[model.DeptUserRelationType]string{}
	for _, r := range relList {
		byType[r.RelationType] = r.DepartmentID
	}
	if byType[model.DeptUserRelationPrimary] != rd.ID || byType[model.DeptUserRelationSecondary] != op.ID {
		t.Fatalf("dept relations mismatch: %+v", byType)
	}

	// 列表：主部门名称回填
	page, err := svc.PageList(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserPageListReq{Name: "svc-pay"})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if page.Total != 1 || page.List[0].MachineUserID != machineID ||
		page.List[0].PrimaryDepartmentID != rd.ID || page.List[0].PrimaryDepartmentName != "研发部" {
		t.Fatalf("page list mismatch: %+v", page)
	}
	// 列表必须同时回传创建时间与更新时间（前端「创建时间」「更新时间」两列直读）
	if item := page.List[0]; item.CreatedAt <= 0 || item.UpdatedAt <= 0 {
		t.Fatalf("createdAt/updatedAt not returned: %+v", item)
	}

	// 详情：部门归属 + 角色
	detail, err := svc.Detail(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserDetailReq{MachineUserID: machineID})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Name != "svc-pay" || len(detail.Roles) != 0 {
		t.Fatalf("detail mismatch: %+v", detail)
	}
	foundPrimary, foundSecondary := false, false
	for _, dept := range detail.Departments {
		switch dept.RelationType {
		case model.DeptUserRelationPrimary:
			foundPrimary = dept.DepartmentID == rd.ID
		case model.DeptUserRelationSecondary:
			foundSecondary = dept.DepartmentID == op.ID
		}
	}
	if !foundPrimary || !foundSecondary {
		t.Fatalf("detail departments mismatch: %+v", detail.Departments)
	}

	// 更新：改主部门→fin、参与部门全量替换为 [op, fin]
	newSecondary := []string{op.ID, fin.ID}
	if err := svc.Update(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserUpdateReq{
		MachineUserID: machineID, Name: "svc-pay-v2", Description: "支付回调V2",
		PrimaryDepartmentID: &fin.ID, SecondaryDepartmentIDs: &newSecondary,
	}); err != nil {
		t.Fatalf("update dept: %v", err)
	}
	relList, err = dao.NewDepartmentUserDao().GetListByCond(newTestTenantCtx(tenantID, adminOp.ID), &dao.DepartmentUserCond{TenantID: tenantID, UserID: machineID})
	if err != nil {
		t.Fatalf("re-query dept relations: %v", err)
	}
	byType = map[model.DeptUserRelationType]string{}
	for _, r := range relList {
		byType[r.RelationType] = r.DepartmentID
	}
	if byType[model.DeptUserRelationPrimary] != fin.ID {
		t.Fatalf("primary should move to fin, got %s", byType[model.DeptUserRelationPrimary])
	}

	// 主部门不可清空
	if err := svc.Update(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserUpdateReq{
		MachineUserID: machineID, Name: "svc-pay-v2", PrimaryDepartmentID: strPtr(""),
	}); err != code.GetError(code.MachineUserDepartmentRequiredError) {
		t.Fatalf("clear primary: want dept required, got %v", err)
	}
	// 挂起
	if err := svc.UpdateStatus(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserStatusReq{MachineUserID: machineID, IsSuspended: true}); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	reloaded, err := dao.NewUserDao().GetByID(newTestTenantCtx(tenantID, adminOp.ID), machineID)
	if err != nil || reloaded == nil {
		t.Fatalf("reload machine user: %v", err)
	}
	if reloaded.Name != "svc-pay-v2" || !reloaded.IsSuspended {
		t.Fatalf("reload mismatch: %+v", reloaded)
	}

	// 普通角色可授、super 角色禁授（按应用授权：role.app_id 须与 req.AppID 一致）
	devRole := &model.RoleEntity{
		TenantID: tenantID, AppID: "app-tenant", Name: "开发者",
		Source: model.RoleSourceCustom, AdminType: model.SysAdminTypeNormal,
	}
	if err := db.Create(devRole).Error; err != nil {
		t.Fatalf("seed dev role: %v", err)
	}
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-tenant", RoleIDs: []string{devRole.ID}}); err != nil {
		t.Fatalf("grant dev role: %v", err)
	}
	adminRole := &model.RoleEntity{
		TenantID: tenantID, AppID: "app-tenant", Name: "租户管理员",
		Source: model.RoleSourceBuiltin, AdminType: model.SysAdminTypeAdmin,
	}
	if err := db.Create(adminRole).Error; err != nil {
		t.Fatalf("seed super role 2: %v", err)
	}
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-tenant", RoleIDs: []string{adminRole.ID}}); err != code.GetError(code.UserAdminRoleAssignForbidden) {
		t.Fatalf("grant super role to machine: want forbidden, got %v", err)
	}

	// 按应用隔离：授另一个应用的普通角色后，原应用角色不受影响；跨应用角色拒绝
	opsRole := &model.RoleEntity{
		TenantID: tenantID, AppID: "app-other", Name: "运维",
		Source: model.RoleSourceCustom, AdminType: model.SysAdminTypeNormal,
	}
	if err := db.Create(opsRole).Error; err != nil {
		t.Fatalf("seed ops role: %v", err)
	}
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-other", RoleIDs: []string{opsRole.ID}}); err != nil {
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
	if err := svc.UpdateRoles(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserRolesUpdateReq{MachineUserID: machineID, AppID: "app-tenant", RoleIDs: []string{opsRole.ID}}); err != code.GetError(code.RoleNotExistError) {
		t.Fatalf("cross-app role grant: want RoleNotExist, got %v", err)
	}

	// 服务账号下有 key 时禁止删除
	key := &model.ApiKeyEntity{
		TenantID: tenantID, OwnerUserID: machineID, Name: "machine-key",
		KeyHash: "h", KeyPrefix: "prefix", Scope: json.RawMessage(`{}`), CreatedBy: adminOp.ID,
	}
	if err := db.Create(key).Error; err != nil {
		t.Fatalf("seed machine key: %v", err)
	}
	if err := svc.Delete(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserDeleteReq{MachineUserID: machineID}); err != code.GetError(code.MachineUserDeleteHasKeysError) {
		t.Fatalf("delete with keys: want has-keys forbidden, got %v", err)
	}
	// 清掉 key 后删除成功：用户软删 + 角色/部门关系级联清理
	if err := dao.NewApiKeyDao().Delete(newTestTenantCtx(tenantID, adminOp.ID), key.ID, adminOp.ID); err != nil {
		t.Fatalf("delete key: %v", err)
	}
	if err := svc.Delete(newTestTenantCtx(tenantID, adminOp.ID), &dtotenant.MachineUserDeleteReq{MachineUserID: machineID}); err != nil {
		t.Fatalf("delete machine user: %v", err)
	}
	gone, err := dao.NewUserDao().GetByID(newTestTenantCtx(tenantID, adminOp.ID), machineID)
	if err != nil {
		t.Fatalf("query deleted machine user: %v", err)
	}
	if gone != nil {
		t.Fatal("machine user should be soft-deleted")
	}
	var deptRelCount int64
	if err := db.Model(&model.DepartmentUserEntity{}).Where("user_id = ?", machineID).Count(&deptRelCount).Error; err != nil {
		t.Fatalf("count dept relations: %v", err)
	}
	if deptRelCount != 0 {
		t.Fatalf("dept relations should be cascade cleaned, got %d", deptRelCount)
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
	testutil.SetupSQLite(t, &model.UserEntity{}, &model.DepartmentEntity{}, &model.DepartmentUserEntity{},
		&model.RoleEntity{}, &model.UserRoleEntity{})
	tenantID := "t1"
	adminOp := seedTestOperator(t, tenantID, true)
	dept := seedTestDept(t, tenantID, "研发部")
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

	svc := NewDepartmentUserSvc()
	ctx := newTestTenantCtx(tenantID, adminOp.ID)
	// leader：拒绝
	if _, err := svc.Create(ctx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: dept.ID,
		UserID:       machine.ID,
		RelationType: model.DeptUserRelationLeader,
	}); err != code.GetError(code.UserMemberOperationOnlyError) {
		t.Fatalf("machine as leader: want member-operation-only, got %v", err)
	}
	// secondary：允许
	if _, err := svc.Create(ctx, &dtotenant.DepartmentUserCreateReq{
		DepartmentID: dept.ID,
		UserID:       machine.ID,
		RelationType: model.DeptUserRelationSecondary,
	}); err != nil {
		t.Fatalf("machine as secondary member: %v", err)
	}
	// 已挂 primary 关系的服务账号收敛为 leader 同样拒绝
	if err := svc.Update(ctx, &dtotenant.DepartmentUserUpdateReq{
		DepartmentID: dept.ID,
		UserID:       machine.ID,
		RelationType: model.DeptUserRelationLeader,
	}); err != code.GetError(code.UserMemberOperationOnlyError) {
		t.Fatalf("converge machine to leader: want member-operation-only, got %v", err)
	}
}
