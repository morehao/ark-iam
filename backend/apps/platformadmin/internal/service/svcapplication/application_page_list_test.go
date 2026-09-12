package svcapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

func newApplicationCtx() *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, "1")
	return ctx
}

// TestApplicationPageListReturnsTimeFields 列表必须同时回传创建时间与更新时间
// （前端「创建时间」「更新时间」两列都读这两个字段，缺失则渲染为 "-"）。
func TestApplicationPageListReturnsTimeFields(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	if err := db.Create(&model.ApplicationEntity{
		Code:   "demo",
		Name:   "演示应用",
		Source: model.AppSourceThirdParty,
	}).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}

	resp, err := NewApplicationSvc().PageList(newApplicationCtx(), &dtoapplication.ApplicationPageListReq{})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("PageList len = %d, want 1", len(resp.List))
	}
	item := resp.List[0]
	if item.CreatedAt <= 0 || item.UpdatedAt <= 0 {
		t.Fatalf("createdAt/updatedAt not returned: %+v", item)
	}
	// source 必须原样回传（前端「来源」列读此字段）
	if item.Source != model.AppSourceThirdParty {
		t.Fatalf("source not returned, got %q", item.Source)
	}
}

// TestApplicationPageListFiltersBySource 按来源过滤只命中同源应用。
func TestApplicationPageListFiltersBySource(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})
	seeds := []model.ApplicationEntity{
		{Code: "admin", Name: "平台管理后台", Source: model.AppSourceBuiltin},
		{Code: "ops-app", Name: "运维自建应用", Source: model.AppSourceFirstParty},
		{Code: "blog", Name: "博客", Source: model.AppSourceThirdParty},
	}
	for i := range seeds {
		if err := db.Create(&seeds[i]).Error; err != nil {
			t.Fatalf("seed application: %v", err)
		}
	}

	svc := NewApplicationSvc()
	for _, tc := range []struct {
		source model.AppSource
		code   string
	}{
		{model.AppSourceBuiltin, "admin"},
		{model.AppSourceFirstParty, "ops-app"},
		{model.AppSourceThirdParty, "blog"},
	} {
		resp, err := svc.PageList(newApplicationCtx(), &dtoapplication.ApplicationPageListReq{Source: tc.source})
		if err != nil {
			t.Fatalf("PageList(%s) failed: %v", tc.source, err)
		}
		if len(resp.List) != 1 || resp.List[0].Code != tc.code {
			t.Fatalf("PageList(%s) = %+v, want only %s", tc.source, resp.List, tc.code)
		}
	}
}

// TestApplicationPageListRejectsIllegalSource 非法来源过滤值必须返回功能级错误码，
// 而不是落成「查不到任何数据」的空列表。
func TestApplicationPageListRejectsIllegalSource(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{})

	resp, err := NewApplicationSvc().PageList(newApplicationCtx(), &dtoapplication.ApplicationPageListReq{
		Source: model.AppSource("foo"),
	})
	if err == nil {
		t.Fatalf("expected error for illegal source, got resp %+v", resp)
	}
}

// TestCreateApplicationForcesThirdParty 控制台创建的应用恒为 third_party：
// source 不由入参决定（DTO 已移除该字段），builtin / first_party 只能由种子与运维产生。
func TestCreateApplicationForcesThirdParty(t *testing.T) {
	testutil.SetupSQLite(t, &model.ApplicationEntity{})

	ctx := newApplicationCtx()
	resp, err := NewApplicationSvc().Create(ctx, &dtoapplication.ApplicationCreateReq{
		Code: "demo",
		Name: "演示应用",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	entity, err := dao.NewApplicationDao().GetByID(ctx, resp.AppID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if entity == nil || entity.ID == "" {
		t.Fatal("created application not found")
	}
	if entity.Source != model.AppSourceThirdParty {
		t.Fatalf("expected source %q, got %q", model.AppSourceThirdParty, entity.Source)
	}
}
