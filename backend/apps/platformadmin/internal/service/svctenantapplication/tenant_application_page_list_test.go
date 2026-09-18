package svctenantapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenantapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

// TestTenantApplicationPageListReturnsTimeFields 列表必须同时回传创建时间与更新时间
// （前端「创建时间」「更新时间」两列都读这两个字段，缺失则渲染为 "-"）。
func TestTenantApplicationPageListReturnsTimeFields(t *testing.T) {
	// 名称回填会查 tenant/application 两张表，故一并注册。
	db := testutil.SetupSQLite(t, &model.TenantApplicationEntity{}, &model.TenantEntity{}, &model.ApplicationEntity{})
	if err := db.Create(&model.TenantApplicationEntity{
		TenantID: "t1",
		AppID:    "app1",
	}).Error; err != nil {
		t.Fatalf("seed tenant application: %v", err)
	}

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "t1")
	ctx.Set(gcontext.KeyUserID, "1")

	resp, err := NewTenantApplicationSvc().PageList(ctx, &dtotenantapplication.TenantApplicationPageListReq{})
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
}
