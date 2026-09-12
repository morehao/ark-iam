package svcapplicationclient

import (
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

// seedApplicationEntity 播种一条应用（列表/详情的所属应用名称来源）。
func seedApplicationEntity(t *testing.T, db *gorm.DB, name string) *model.ApplicationEntity {
	t.Helper()
	app := &model.ApplicationEntity{
		Code:   "app_code_" + name,
		Name:   name,
		Source: model.AppSourceBuiltin,
		Status: model.AppStatusEnable,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("seed application %s: %v", name, err)
	}
	return app
}

// TestApplicationClientPageListReturnsAppName 列表/详情都要回填所属应用名称。
// 回归：只回传 appID 时前端「所属应用」列只能显示 UUID，可读性差（用户无法辨认客户端归属）。
func TestApplicationClientPageListReturnsAppName(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	app := seedApplicationEntity(t, db, "应用一")

	client := newTestClientEntity("客户端1", "client-1", model.ApplicationClientSourceThirdParty)
	client.AppID = app.ID
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("seed application client: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "u1")
	svc := NewApplicationClientSvc()

	resp, err := svc.PageList(ctx, &dtoapplicationclient.ApplicationClientPageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("PageList: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("列表条数 = %d, want 1", len(resp.List))
	}
	if resp.List[0].AppID != app.ID {
		t.Fatalf("appID = %q, want %q", resp.List[0].AppID, app.ID)
	}
	if resp.List[0].AppName != "应用一" {
		t.Fatalf("appName = %q, want %q", resp.List[0].AppName, "应用一")
	}

	detail, err := svc.Detail(ctx, &dtoapplicationclient.ApplicationClientDetailReq{ApplicationClientID: client.ID})
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if detail.AppName != "应用一" {
		t.Fatalf("详情 appName = %q, want %q", detail.AppName, "应用一")
	}
	// 时间列契约（列表页规范 R3）：创建时间与更新时间都必须回传
	if resp.List[0].CreatedAt <= 0 || resp.List[0].UpdatedAt <= 0 {
		t.Fatalf("createdAt/updatedAt 必须 > 0, got %d/%d", resp.List[0].CreatedAt, resp.List[0].UpdatedAt)
	}
}

// TestApplicationClientAppNameEmptyWhenAppMissing 应用行缺失（历史脏数据/跨租户引用）时留空名，
// 不因展示性字段让整个列表报错，前端退化为展示应用 ID。
func TestApplicationClientAppNameEmptyWhenAppMissing(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	client := newTestClientEntity("客户端1", "client-1", model.ApplicationClientSourceThirdParty)
	client.AppID = "no-such-app"
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("seed application client: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")

	resp, err := NewApplicationClientSvc().PageList(ctx, &dtoapplicationclient.ApplicationClientPageListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("PageList: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("列表条数 = %d, want 1", len(resp.List))
	}
	if resp.List[0].AppName != "" {
		t.Fatalf("悬空 appID 应留空名, got %q", resp.List[0].AppName)
	}
}
