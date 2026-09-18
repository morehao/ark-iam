package svctenant

import (
	"encoding/json"
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

// seedTestRole 播种**空编码**的自建角色（新模型下租户自建角色的常态形态）。
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

	// 非法应用（非本租户订阅的应用）
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app-bad", Name: "管理员"}); err == nil {
		t.Fatalf("expected invalid app error")
	}
	// 创建成功
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Name: "管理员"}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	// 同应用名称唯一
	if _, err := svc.Create(ginCtx, &dtotenant.RoleCreateReq{AppID: "app1", Name: "管理员"}); gerror.GetCode(err) != int(code.RoleCreateError) {
		t.Fatalf("expected duplicate name error, got %v", err)
	}
	// 其他租户同名不冲突（名称唯一性在「租户 × 应用」维度）
	if _, err := svc.Create(newDeptGinCtx(t, "t2", "op2"), &dtotenant.RoleCreateReq{AppID: "app2", Name: "管理员"}); err != nil {
		t.Fatalf("cross-tenant same name should be allowed: %v", err)
	}
}

// TestRoleCreateCannotInjectCode 租户侧在**结构上**没有写入角色编码的入口：请求体里即使带了 code，
// 绑定层也没有字段接收它，落库的自建角色编码恒为空串。这是"契约值归应用方"的编译期 + 运行期双重保证
// ——自建角色因此不可能与模板角色/产品锚点撞码，也不可能自造一个值去命中下游已有策略。
func TestRoleCreateCannotInjectCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestApp(t, db, "t1", "app1")

	ginCtx := newDeptGinCtx(t, "t1", "op")
	// 直接喂 JSON：模拟前端/裸调 API 试图夹带契约值
	var req dtotenant.RoleCreateReq
	if err := json.Unmarshal([]byte(`{"appID":"app1","name":"管理员","code":"tenant_admin"}`), &req); err != nil {
		t.Fatalf("bind req: %v", err)
	}
	resp, err := svc.Create(ginCtx, &req)
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	detail, err := svc.Detail(ginCtx, &dtotenant.RoleDetailReq{RoleID: resp.RoleID})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Code != "" {
		t.Fatalf("自建角色编码必须为空串，实际 %q（租户不得写入契约值）", detail.Code)
	}
	if detail.Source != model.RoleSourceCustom {
		t.Fatalf("source = %s, want custom", detail.Source)
	}
}

// TestRoleUpdateBuiltinForbidden 内置角色整体只读：模板角色（应用角色模板物化）与产品锚点的编码/名称
// 都由应用方定义，租户不得改名——权威判定必须在服务层（前端置灰只是 UX 兜底），
// 且不得因为「字段原样回传」而放行整条更新。
func TestRoleUpdateBuiltinForbidden(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestApp(t, db, "t1", "app1")
	// 产品锚点角色
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
	// 模板角色（由应用角色模板物化的普通内置角色）
	if err := db.Create(&model.RoleEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "rt"}},
		TenantID:   "t1",
		AppID:      "app1",
		Code:       "storage_admin",
		Name:       "存储管理员",
		Source:     model.RoleSourceBuiltin,
		AdminType:  model.SysAdminTypeNormal,
		CreatedBy:  "t",
	}).Error; err != nil {
		t.Fatalf("seed template role: %v", err)
	}

	ginCtx := newDeptGinCtx(t, "t1", "op")
	// 改名与「原样回传」都必须拒绝：只读语义与内容是否变化无关
	for _, req := range []*dtotenant.RoleUpdateReq{
		{RoleID: "rb", Name: "改名试试"},
		{RoleID: "rb", Name: "租户管理员"},
		{RoleID: "rt", Name: "存储超管"},
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

// TestRoleUpdateCustomRoleKeepsCode 自建角色的更新只作用于名称/描述：编码不在入参里，
// 更新既不会写入也不会清掉存量行的编码（存量行可能带着历史编码）。
func TestRoleUpdateCustomRoleKeepsCode(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RoleMenuEntity{},
		&model.DepartmentEntity{}, &model.DepartmentUserEntity{}, &model.ApplicationEntity{}, &model.TenantApplicationEntity{})
	svc := &roleSvc{}
	seedTenantAdminOperator(t, db, "t1", "op")
	seedTestApp(t, db, "t1", "app1")
	seedTestRoleWithCode(t, db, "r1", "t1", "app1", "管理员", "custom_admin")
	seedTestRoleWithCode(t, db, "r2", "t1", "app1", "只读", "storage_readonly")

	ginCtx := newDeptGinCtx(t, "t1", "op")

	// 与他人重名拒绝
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Name: "只读"}); gerror.GetCode(err) != int(code.RoleUpdateError) {
		t.Fatalf("expected duplicate name error, got %v", err)
	}
	// 自身同名放行（唯一性校验排除自身）
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Name: "管理员"}); err != nil {
		t.Fatalf("update with unchanged name: %v", err)
	}
	// 改名生效，编码保持不变
	if err := svc.Update(ginCtx, &dtotenant.RoleUpdateReq{RoleID: "r1", Name: "存储管理员", Description: "负责存储"}); err != nil {
		t.Fatalf("update name: %v", err)
	}
	detail, err := svc.Detail(ginCtx, &dtotenant.RoleDetailReq{RoleID: "r1"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Name != "存储管理员" || detail.Description != "负责存储" {
		t.Fatalf("name/description not updated, got %q/%q", detail.Name, detail.Description)
	}
	if detail.Code != "custom_admin" {
		t.Fatalf("encode must stay untouched by update, got %q", detail.Code)
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
