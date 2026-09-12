package svcapplicationclient

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// TestUpdateBuiltInClientRejectsSeedOwnedName 内置 OAuth 客户端的名称是种子收敛字段
// （字段权威矩阵 reconcile），控制台必须拒写；回调地址等运行参数是 create_only（归运维），可改。
func TestUpdateBuiltInClientRejectsSeedOwnedName(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{})
	builtin := &model.ApplicationClientEntity{
		TenantID: "1", AppID: "app1", Code: "platform-admin-web", Name: "平台管理后台",
		Source: model.ApplicationClientSourceBuiltin, Status: model.ApplicationClientStatusEnable,
	}
	if err := db.Create(builtin).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "0")
	svc := NewApplicationClientSvc()

	err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: builtin.ID,
		Name:                "运维自定名",
		Status:              model.ApplicationClientStatusEnable,
	})
	if err == nil {
		t.Fatal("内置客户端改名必须被拒绝")
	}
	if gerror.GetCode(err) != int(code.ApplicationClientBuiltInFieldImmutableError) {
		t.Fatalf("期望 ApplicationClientBuiltInFieldImmutableError, got %v", err)
	}
	got, err := dao.NewApplicationClientDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "平台管理后台" {
		t.Fatalf("被拒的改名不得落库, got %q", got.Name)
	}

	// 运行参数归运维：名称不变时改回调地址应放行
	if err := svc.Update(ctx, &dtoapplicationclient.ApplicationClientUpdateReq{
		ApplicationClientID: builtin.ID,
		Name:                builtin.Name,
		Status:              model.ApplicationClientStatusEnable,
		RedirectURIs:        []string{"https://sso.example.com/auth/callback"},
	}); err != nil {
		t.Fatalf("内置客户端运行参数应可改: %v", err)
	}
	got, err = dao.NewApplicationClientDao().GetByID(ctx, builtin.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if string(got.RedirectURIs) == "" || string(got.RedirectURIs) == "[]" {
		t.Fatalf("回调地址未落库: %s", got.RedirectURIs)
	}
}
