package svctenant

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gerror"
	"gorm.io/gorm"
)

func seedTestApp(t *testing.T, db *gorm.DB, tenantID, appID string) {
	t.Helper()
	if err := db.Create(&model.ApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: appID}},
		Code:       "app-" + appID,
		Name:       "租户管理后台",
		Status:     "enable",
	}).Error; err != nil {
		t.Fatalf("seed app: %v", err)
	}
	if err := db.Create(&model.TenantApplicationEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ta-" + appID}},
		TenantID:     tenantID,
		AppID:        appID,
		Status:       "enable",
		Config:       []byte("{}"),
		GrantedScope: []byte("[]"),
	}).Error; err != nil {
		t.Fatalf("seed tenant application: %v", err)
	}
}

func seedTestRole(t *testing.T, db *gorm.DB, id, tenantID, appID, name string) {
	t.Helper()
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		AppID:      appID,
		Name:       name,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
}

// seedTestRoleWithCode 播种带编码的角色（编码类断言专用；seedTestRole 保持"空编码存量行"形态）。
func seedTestRoleWithCode(t *testing.T, db *gorm.DB, id, tenantID, appID, name string, roleCode model.RoleCode) {
	t.Helper()
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}},
		TenantID:   tenantID,
		AppID:      appID,
		Code:       roleCode,
		Name:       name,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
}

func seedTestMenuTree(t *testing.T, db *gorm.DB, tenantID string) (rootID, childID string) {
	t.Helper()
	seedTestApp(t, db, tenantID, "app1")
	root := &model.MenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "m1"}},
		AppID:      "app1",
		Name:       "部门架构",
		Code:       "department",
		Path:       "/department",
		Status:     "enable",
	}
	child := &model.MenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "m2"}},
		AppID:      "app1",
		ParentID:   "m1",
		Name:       "用户管理",
		Code:       "tenant-user",
		Path:       "/user",
		Status:     "enable",
	}
	for _, m := range []*model.MenuEntity{root, child} {
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("seed menu: %v", err)
		}
	}
	return "m1", "m2"
}

// TestRoleCreateRequiresApp 角色从属于租户订阅的应用：非法应用拒绝、应用内名称唯一、跨应用/租户同名允许。
func TestRoleCreateRequiresApp(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestApp(t, db, "t1", "app1")
	seedTestApp(t, db, "t2", "app2")
	seedTenantAdminOperator(t, db, "t2", "op2")

	ginCtx := newDeptGinCtx(t, "t1", "op")

	// 非法应用（编码合法且非保留，确保失败原因是应用而非编码）
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app-bad", Code: "custom_admin", Name: "管理员"}); err == nil {
		t.Fatalf("expected invalid app error")
	}
	// 创建成功
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Code: "custom_admin", Name: "管理员"}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	// 同应用名称唯一（换编码，确保失败原因是名称而非编码）
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Code: "other_admin", Name: "管理员"}); err == nil {
		t.Fatalf("expected duplicate name error")
	}
	// 其他租户同名同码都不冲突（编码唯一性是租户维度）
	if _, err := svc.Create(newDeptGinCtx(t, "t2", "op2"), &dtotenant.RoleCreateReq{AppID: "app2", Code: "custom_admin", Name: "管理员"}); err != nil {
		t.Fatalf("cross-tenant same name/code should be allowed: %v", err)
	}
}

// TestRoleCreateValidatesCode 角色编码是跨系统授权契约值（OIDC groups 取值）：
// 形状白名单 + 租户内唯一，均属可预期业务边界，必须返回各自的专用错误码。
func TestRoleCreateValidatesCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTenantAdminOperator(t, db, "t2", "op2")
	seedTestApp(t, db, "t1", "app1")
	seedTestApp(t, db, "t2", "app2")

	ginCtx := newDeptGinCtx(t, "t1", "op")

	// 形状非法：空串 / 大写 / 数字开头 / 连字符 / 下划线开头 / 空格
	for _, badCode := range []model.RoleCode{"", "Platform_Admin", "1admin", "platform-admin", "_admin", "platform admin"} {
		_, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Code: badCode, Name: "角色" + string(badCode)})
		if gerror.GetCode(err) != int(code.RoleCodeInvalidError) {
			t.Fatalf("expected invalid code error for %q, got %v", badCode, err)
		}
	}

	// 合法编码
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Code: "custom_admin", Name: "管理员"}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	// 同租户重码拒绝：下游只看到编码，跨应用重码同样产生授权歧义
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Code: "custom_admin", Name: "另一个角色"}); gerror.GetCode(err) != int(code.RoleCodeExistsError) {
		t.Fatalf("expected duplicate code error, got %v", err)
	}
	// 跨租户同码允许
	if _, err := svc.Create(newDeptGinCtx(t, "t2", "op2"), &dtotenant.RoleCreateReq{AppID: "app2", Code: "custom_admin", Name: "管理员"}); err != nil {
		t.Fatalf("cross-tenant same code should be allowed: %v", err)
	}
}

// TestRoleCreateReservedCodeForbidden 系统保留编码（内置角色的下游策略锚点）不得被自建角色占用：
// 下游按「前缀 + 编码」认策略名，前缀隔离不了同码，放开即等于允许任何租户自造一个撞上内置策略的编码。
func TestRoleCreateReservedCodeForbidden(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestApp(t, db, "t1", "app1")

	ginCtx := newDeptGinCtx(t, "t1", "op")
	// 普通租户里内置 platform_admin 并不存在（只随种子/平台租户出现），租户内唯一性校验会放行，必须由保留字拦住
	for _, reserved := range []model.RoleCode{model.RoleCodePlatformAdmin, model.RoleCodeTenantAdmin} {
		_, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Code: reserved, Name: "伪装角色"})
		if gerror.GetCode(err) != int(code.RoleCodeReservedError) {
			t.Fatalf("expected reserved code error for %q, got %v", reserved, err)
		}
	}
	// 非保留编码不受影响
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Code: "storage_readonly", Name: "只读"}); err != nil {
		t.Fatalf("non-reserved code should be allowed: %v", err)
	}
	// 存量行（本守卫上线前已占用保留码的自建角色）仍可改名称/描述：只在把编码改成保留码时拦截
	seedTestRoleWithCode(t, db, "legacy", "t1", "app1", "历史角色", model.RoleCodePlatformAdmin)
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "legacy", Code: model.RoleCodePlatformAdmin, Name: "改名"}); err != nil {
		t.Fatalf("legacy row with unchanged reserved code should allow name update: %v", err)
	}
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "legacy", Code: model.RoleCodeTenantAdmin, Name: "改名"}); gerror.GetCode(err) != int(code.RoleCodeReservedError) {
		t.Fatalf("expected reserved code error when renaming into reserved code, got %v", err)
	}
}

// TestRoleUpdateCode 编码可改（不是本系统的定位键，改错可在本控制台改回），
// 但改动即改变下游授权：形状与租户内唯一必须守住，且列表/详情要回传最新值。
func TestRoleUpdateCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestApp(t, db, "t1", "app1")
	seedTestRoleWithCode(t, db, "r1", "t1", "app1", "管理员", "custom_admin")
	seedTestRoleWithCode(t, db, "r2", "t1", "app1", "只读", "storage_readonly")

	ginCtx := newDeptGinCtx(t, "t1", "op")

	// 非法形状
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Code: "Platform", Name: "管理员"}); gerror.GetCode(err) != int(code.RoleCodeInvalidError) {
		t.Fatalf("expected invalid code error, got %v", err)
	}
	// 与他人重码
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Code: "storage_readonly", Name: "管理员"}); gerror.GetCode(err) != int(code.RoleCodeExistsError) {
		t.Fatalf("expected duplicate code error, got %v", err)
	}
	// 改名为系统保留编码拒绝（与创建同一条提权路径）
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Code: model.RoleCodeTenantAdmin, Name: "管理员"}); gerror.GetCode(err) != int(code.RoleCodeReservedError) {
		t.Fatalf("expected reserved code error, got %v", err)
	}
	// 编码不变（唯一性校验排除自身）应放行
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Code: "custom_admin", Name: "管理员"}); err != nil {
		t.Fatalf("update with unchanged code: %v", err)
	}
	// 改码生效并回传
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Code: "console_admin", Name: "管理员"}); err != nil {
		t.Fatalf("update code: %v", err)
	}
	detail, err := svc.Detail(ginCtx, &dtotenant.RoleDetailReq{RoleID: "r1"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Code != "console_admin" {
		t.Fatalf("expected updated code console_admin, got %q", detail.Code)
	}
}

// TestRoleUpdateBuiltinForbidden 内置角色整体只读：编码是下游策略供给锚点，名称/描述由种子与运维负责。
// 权威判定必须在服务层（前端置灰只是 UX 兜底），且不得因为「编码没变」而放行整条更新。
func TestRoleUpdateBuiltinForbidden(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestApp(t, db, "t1", "app1")
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "rb"}},
		TenantID:   "t1",
		AppID:      "app1",
		Code:       model.RoleCodeTenantAdmin,
		Name:       "租户管理员",
		Source:     model.RoleSourceBuiltin,
		AdminType:  model.SysAdminTypeAdmin,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed builtin role: %v", err)
	}

	ginCtx := newDeptGinCtx(t, "t1", "op")
	// 改码、改名称、以及「三字段原样回传」都必须拒绝：只读语义与内容是否变化无关
	for _, req := range []*dtotenant.RoleUpdateReq{
		{RoleID: "rb", Code: "storage_readonly", Name: "租户管理员"},
		{RoleID: "rb", Code: model.RoleCodeTenantAdmin, Name: "改名试试"},
		{RoleID: "rb", Code: model.RoleCodeTenantAdmin, Name: "租户管理员"},
	} {
		if err := svc.Update(ginCtx, req); gerror.GetCode(err) != int(code.RoleUpdateBuiltinForbiddenError) {
			t.Fatalf("expected builtin-forbidden error for %+v, got %v", req, err)
		}
	}

	detail, err := svc.Detail(ginCtx, &dtotenant.RoleDetailReq{RoleID: "rb"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Code != model.RoleCodeTenantAdmin || detail.Name != "租户管理员" {
		t.Fatalf("builtin role must stay untouched, got code=%q name=%q", detail.Code, detail.Name)
	}
}

// TestRolePageListReturnsCode 列表回传编码（前端「编码」列直读），且关键词同时匹配名称与编码。
func TestRolePageListReturnsCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestRoleWithCode(t, db, "r1", "t1", "app1", "平台管理员", "platform_admin")
	seedTestRoleWithCode(t, db, "r2", "t1", "app1", "对象存储只读", "storage_readonly")

	ginCtx := newDeptGinCtx(t, "t1", "op")

	resp, err := svc.PageList(ginCtx, &dtotenant.RolePageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	codeMap := make(map[string]model.RoleCode, len(resp.List))
	for _, item := range resp.List {
		codeMap[item.RoleID] = item.Code
	}
	if codeMap["r1"] != "platform_admin" || codeMap["r2"] != "storage_readonly" {
		t.Fatalf("expected codes returned in list, got %+v", resp.List)
	}

	// 关键词命中编码（名称里没有 "storage"）
	hit, err := svc.PageList(ginCtx, &dtotenant.RolePageListReq{Page: 1, PageSize: 10, Keyword: "storage"})
	if err != nil {
		t.Fatalf("page list by keyword: %v", err)
	}
	if hit.Total != 1 || hit.List[0].RoleID != "r2" {
		t.Fatalf("expected keyword to match code, got total=%d list=%+v", hit.Total, hit.List)
	}
}

// TestRolePageListWithCounts 分页列表带成员数/菜单数聚合。
func TestRolePageListWithCounts(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestRole(t, db, "r1", "t1", "app1", "管理员")

	if err := db.Create(&model.UserRoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "ur1"}},
		TenantID:   "t1",
		UserID:     "u1",
		RoleID:     "r1",
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed user_role: %v", err)
	}
	if err := db.Create(&model.RoleMenuEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "rm1"}},
		TenantID:   "t1",
		RoleID:     "r1",
		MenuID:     "m1",
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed role_menu: %v", err)
	}

	resp, err := svc.PageList(newDeptGinCtx(t, "t1", "op"), &dtotenant.RolePageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
	item := resp.List[0]
	if item.MemberCount != 1 || item.MenuCount != 1 {
		t.Fatalf("unexpected counts: member=%d menu=%d", item.MemberCount, item.MenuCount)
	}
	if item.AppID != "app1" {
		t.Fatalf("expected appID app1, got %s", item.AppID)
	}
	// 列表必须同时回传创建时间与更新时间（前端「创建时间」「更新时间」两列直读）
	if item.CreatedAt <= 0 || item.UpdatedAt <= 0 {
		t.Fatalf("createdAt/updatedAt not returned: %+v", item)
	}
}

// TestRolePageListUnassigned 未归属应用的系统角色（app_id 为空串）必须能由服务端过滤：
// 前端「系统角色」下拉若靠"全量拉一页再客户端过滤"，租户角色超过一页时会漏掉未归属角色。
func TestRolePageListUnassigned(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTestApp(t, db, "t1", "app1")
	seedTestRole(t, db, "r1", "t1", "app1", "应用管理员")
	seedTestRole(t, db, "r2", "t1", "", "系统管理员")

	resp, err := svc.PageList(newDeptGinCtx(t, "t1", "op"), &dtotenant.RolePageListReq{Page: 1, PageSize: 10, Unassigned: true})
	if err != nil {
		t.Fatalf("page list unassigned: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].RoleID != "r2" {
		t.Fatalf("expected only unassigned role r2, got total=%d list=%+v", resp.Total, resp.List)
	}

	// 不传 unassigned 时不过滤（空串 appID 语义是"不过滤"）
	all, err := svc.PageList(newDeptGinCtx(t, "t1", "op"), &dtotenant.RolePageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("page list all: %v", err)
	}
	if all.Total != 2 {
		t.Fatalf("expected 2 roles without filter, got %d", all.Total)
	}

	// 与关键词组合：未归属 + 名称模糊命中
	hit, err := svc.PageList(newDeptGinCtx(t, "t1", "op"), &dtotenant.RolePageListReq{Page: 1, PageSize: 10, Unassigned: true, Keyword: "系统"})
	if err != nil {
		t.Fatalf("page list unassigned+keyword: %v", err)
	}
	if hit.Total != 1 || hit.List[0].RoleID != "r2" {
		t.Fatalf("expected r2 for unassigned+keyword, got total=%d list=%+v", hit.Total, hit.List)
	}
}

// TestRoleMenusUpdateAndGet 角色菜单授权：按角色所属应用菜单授权 + 回显 + 非法菜单拒绝。
func TestRoleMenusUpdateAndGet(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.RoleMenuEntity{}, &model.MenuEntity{}, &model.UserRoleEntity{},
		&model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestRole(t, db, "r1", "t1", "app1", "管理员")
	rootID, childID := seedTestMenuTree(t, db, "t1")

	ginCtx := newDeptGinCtx(t, "t1", "op")

	// 非法菜单（不在角色所属应用菜单集合内）
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r1", MenuIDs: []string{"m-bad"}}); err == nil {
		t.Fatalf("expected invalid menu error")
	}

	// 全量替换
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r1", MenuIDs: []string{rootID, childID}}); err != nil {
		t.Fatalf("update menus: %v", err)
	}

	// 回显
	resp, err := svc.GetMenus(ginCtx, &dtotenant.RoleDetailReq{RoleID: "r1"})
	if err != nil {
		t.Fatalf("get menus: %v", err)
	}
	if len(resp.List) != 1 || len(resp.List[0].Children) != 1 {
		t.Fatalf("unexpected tree: %+v", resp.List)
	}
	if len(resp.MenuIDs) != 2 {
		t.Fatalf("expected 2 authorized menu ids, got %+v", resp.MenuIDs)
	}

	// 全量替换为空（撤销全部授权）
	if err := svc.UpdateMenus(ginCtx, &dtotenant.RoleMenusUpdateReq{RoleID: "r1", MenuIDs: []string{}}); err != nil {
		t.Fatalf("clear menus: %v", err)
	}
	resp, err = svc.GetMenus(ginCtx, &dtotenant.RoleDetailReq{RoleID: "r1"})
	if err != nil {
		t.Fatalf("get menus: %v", err)
	}
	if len(resp.MenuIDs) != 0 {
		t.Fatalf("expected cleared menu ids, got %+v", resp.MenuIDs)
	}
}
