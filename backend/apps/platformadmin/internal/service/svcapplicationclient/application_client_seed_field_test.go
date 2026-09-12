package svcapplicationclient

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

// TestUpdateBuiltInClientAllowsName 内置 OAuth 客户端的名称归运维（字段权威矩阵 create_only）：
// 控制台可改且重启不被种子回写；回调地址等运行参数一直归运维。
//
// 回归背景：名称一度是 reconcile，控制台改名直接报"不可修改"；reconcile 收窄到
// 「定位键 + 安全不变式」后（客户端只剩 source/app_id），名称交还运维。
func TestUpdateBuiltInClientAllowsName(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	builtin := &model.ApplicationClientEntity{
		TenantID: "1", AppID: "app1", Code: model.SeedBuiltinClientPlatformAdminWeb, Name: "平台管理后台",
		Source: model.ApplicationClientSourceBuiltin, Status: model.ApplicationClientStatusEnable,
	}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewApplicationClientSvc()

	// 改名 + 改回调地址：一并放行
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: builtin.ID,
		Name:                "运维自定名",
		Status:              model.ApplicationClientStatusEnable,
		RedirectURIs:        []string{"https://sso.example.com/auth/callback"},
	}); err != nil {
		t.Fatalf("内置客户端展示字段应可改: %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "运维自定名" {
		t.Fatalf("改名未落库, got %q", got.Name)
	}
	if string(got.RedirectURIs) == "" || string(got.RedirectURIs) == "[]" {
		t.Fatalf("回调地址未落库: %s", got.RedirectURIs)
	}
	// source（安全不变式）不在 Update 请求里，控制台无法改写内置标记
	if got.Source != model.ApplicationClientSourceBuiltin {
		t.Fatalf("source 不得被控制台改写: %q", got.Source)
	}
}
