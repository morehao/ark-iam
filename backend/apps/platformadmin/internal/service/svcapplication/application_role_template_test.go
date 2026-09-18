package svcapplication

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/gerror"
)

// TestApplicationRoleTemplateValidation 应用角色模板是契约值的唯一写入入口，形状必须在这里守住：
// 编码形状、模板内重码、名称空/超长、产品锚点编码、条目数超限一律拒（ApplicationRoleTemplateInvalidError）。
func TestApplicationRoleTemplateValidation(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.RoleEntity{}, &model.TenantApplicationEntity{})

	ctx := newApplicationCtx()
	svc := NewApplicationSvc()

	tooMany := make([]model.RoleTemplateItem, 0, 65)
	for i := 0; i < 65; i++ {
		tooMany = append(tooMany, model.RoleTemplateItem{Code: model.RoleCode("role_" + strings.Repeat("a", i%5) + string(rune('a'+i%26)) + string(rune('a'+i/26))), Name: "角色"})
	}
	cases := map[string][]model.RoleTemplateItem{
		"编码形状非法": {{Code: "Console_Admin", Name: "控制台管理员"}},
		"名称空白":   {{Code: "readonly", Name: "   "}},
		"模板内重码":  {{Code: "readonly", Name: "只读"}, {Code: "readonly", Name: "只读用户"}},
		"产品锚点编码": {{Code: model.RoleCodeTenantAdmin, Name: "租户管理员"}},
		"名称超长":   {{Code: "readonly", Name: strings.Repeat("名", 129)}},
		"条目数超上限": tooMany,
	}
	for name, items := range cases {
		_, err := svc.Create(ctx, &dtoapplication.ApplicationCreateReq{
			Code:         "app_bad",
			Name:         "非法模板",
			RoleTemplate: items,
		})
		if gerror.GetCode(err) != int(code.ApplicationRoleTemplateInvalidError) {
			t.Fatalf("%s: err = %v, want code %d", name, err, code.ApplicationRoleTemplateInvalidError)
		}
	}

	// 合法模板：名称自动去空白后落库
	resp, err := svc.Create(ctx, &dtoapplication.ApplicationCreateReq{
		Code: "app_ok",
		Name: "对象存储",
		RoleTemplate: []model.RoleTemplateItem{
			{Code: "storage_admin", Name: " 存储管理员 "},
			{Code: "readonly", Name: "只读用户"},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	got := detailRoleTemplate(t, svc, ctx, resp.AppID)
	if len(got) != 2 || got[0].Code != "storage_admin" || got[0].Name != "存储管理员" || got[1].Code != "readonly" {
		t.Fatalf("roleTemplate = %v, want [{storage_admin 存储管理员} {readonly 只读用户}]", got)
	}
}

// TestApplicationRoleTemplateUpdateAndFanOut 模板更新的三种语义（null 不修改 / [] 清空 / 传值全量替换）
// 必须真实物化到**已订阅该应用的租户**：模板新增 → 租户侧出现 builtin 角色；模板移除 → 该角色被撤下。
// 少了这条 fan-out，平台改模板后存量租户的角色集合会永久漂移（模板说一套、租户里是另一套）。
func TestApplicationRoleTemplateUpdateAndFanOut(t *testing.T) {
	db := testutil.SetupSQLite(t,
		&model.ApplicationEntity{}, &model.RoleEntity{}, &model.TenantApplicationEntity{},
		&model.UserRoleEntity{}, &model.RoleMenuEntity{},
	)

	ctx := newApplicationCtx()
	svc := NewApplicationSvc()

	resp, err := svc.Create(ctx, &dtoapplication.ApplicationCreateReq{
		Code:         "app_store",
		Name:         "对象存储",
		RoleTemplate: []model.RoleTemplateItem{{Code: "storage_admin", Name: "存储管理员"}},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	// 两个租户已订阅该应用（在模板变更之前就订阅了）
	for _, tenantID := range []string{"t1", "t2"} {
		if err := db.Create(&model.TenantApplicationEntity{
			TenantID: tenantID, AppID: resp.AppID, Status: model.TenantApplicationStatusEnable,
			Config: datatypes.JSON("{}"), GrantedScope: datatypes.JSON("[]"),
		}).Error; err != nil {
			t.Fatalf("seed subscription %s: %v", tenantID, err)
		}
	}

	// null 表示不修改：不传该字段时既不清空也不追加，且不应触发物化
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{AppID: resp.AppID, Name: "对象存储"}); err != nil {
		t.Fatalf("Update without roleTemplate failed: %v", err)
	}
	if got := detailRoleTemplate(t, svc, ctx, resp.AppID); len(got) != 1 {
		t.Fatalf("roleTemplate should be kept when request omits it, got %v", got)
	}
	if n := countRoles(t, ctx, resp.AppID); n != 0 {
		t.Fatalf("roles should not be materialized when template untouched, got %d", n)
	}

	// 传值即全量替换：模板落库 + 物化到两个已订阅租户
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID: resp.AppID,
		Name:  "对象存储",
		RoleTemplate: []model.RoleTemplateItem{
			{Code: "storage_admin", Name: "存储管理员"},
			{Code: "readonly", Name: "只读用户"},
		},
	}); err != nil {
		t.Fatalf("Update with roleTemplate failed: %v", err)
	}
	if got := detailRoleTemplate(t, svc, ctx, resp.AppID); len(got) != 2 {
		t.Fatalf("roleTemplate = %v, want 2 items", got)
	}
	for _, tenantID := range []string{"t1", "t2"} {
		roles := listRoles(t, ctx, tenantID, resp.AppID)
		if len(roles) != 2 {
			t.Fatalf("tenant %s roles = %d, want 2 (materialized from template)", tenantID, len(roles))
		}
		for _, role := range roles {
			if role.Source != model.RoleSourceBuiltin {
				t.Fatalf("tenant %s role %s source = %s, want builtin", tenantID, role.Code, role.Source)
			}
		}
	}

	// 模板移除 readonly：两个租户下的该角色被撤下（连带清理用户/菜单授权）
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:        resp.AppID,
		Name:         "对象存储",
		RoleTemplate: []model.RoleTemplateItem{{Code: "storage_admin", Name: "存储管理员"}},
	}); err != nil {
		t.Fatalf("Update removing template item failed: %v", err)
	}
	for _, tenantID := range []string{"t1", "t2"} {
		roles := listRoles(t, ctx, tenantID, resp.AppID)
		if len(roles) != 1 || roles[0].Code != "storage_admin" {
			t.Fatalf("tenant %s roles = %v, want only storage_admin", tenantID, roles)
		}
	}

	// [] 表示清空：模板清空后租户侧的模板角色全部撤下
	if err := svc.Update(ctx, &dtoapplication.ApplicationUpdateReq{
		AppID:        resp.AppID,
		Name:         "对象存储",
		RoleTemplate: []model.RoleTemplateItem{},
	}); err != nil {
		t.Fatalf("Update with empty roleTemplate failed: %v", err)
	}
	if got := detailRoleTemplate(t, svc, ctx, resp.AppID); len(got) != 0 {
		t.Fatalf("roleTemplate should be cleared by [], got %v", got)
	}
	if n := countRoles(t, ctx, resp.AppID); n != 0 {
		t.Fatalf("roles should be withdrawn after clearing template, got %d", n)
	}
}

// detailRoleTemplate 读取应用详情里的角色模板（断言用）。
func detailRoleTemplate(t *testing.T, svc ApplicationSvc, ctx *gin.Context, appID string) []model.RoleTemplateItem {
	t.Helper()
	detail, err := svc.Detail(ctx, &dtoapplication.ApplicationDetailReq{AppID: appID})
	if err != nil {
		t.Fatalf("Detail(%s) failed: %v", appID, err)
	}
	return detail.RoleTemplate
}

// listRoles 跨租户读取指定租户在某应用下的角色（role 是租户表，测试里显式声明跨租户作用域）。
func listRoles(t *testing.T, ctx *gin.Context, tenantID, appID string) []model.RoleEntity {
	t.Helper()
	roles, err := dao.NewRoleDao().GetListByCond(dbclient.ExplicitTenantContext(ctx, tenantID), &dao.RoleCond{AppID: appID})
	if err != nil {
		t.Fatalf("list roles of tenant %s failed: %v", tenantID, err)
	}
	return roles
}

// countRoles 跨租户统计某应用下的角色总数（断言"模板清空后全部撤下"）。
func countRoles(t *testing.T, ctx *gin.Context, appID string) int {
	t.Helper()
	roles, err := dao.NewRoleDao().GetListByCond(dbclient.CrossTenantContext(ctx), &dao.RoleCond{AppID: appID})
	if err != nil {
		t.Fatalf("count roles of app %s failed: %v", appID, err)
	}
	return len(roles)
}
