package svcapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// newDeleteCtx 构造带租户与操作人上下文的 gin.Context。
func newDeleteCtx(userID uint) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, uint(1))
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

func TestDeleteSystemApplication(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})

	entity := &model.ApplicationEntity{
		Code:     "admin",
		Name:     "管理后台",
		IsSystem: 1,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewApplicationSvc()
	err := svc.Delete(newDeleteCtx(0), &dtoapplication.ApplicationDeleteReq{AppID: entity.ID})
	if err == nil {
		t.Fatal("expected error for system-built-in application")
	}
	gerr, ok := err.(*gerror.Error)
	if !ok || gerr.Code != int(code.ApplicationSystemBuiltInErr) {
		t.Fatalf("expected ApplicationSystemBuiltInErr, got %v", err)
	}
}

func TestDeleteNonSystemApplication(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})

	entity := &model.ApplicationEntity{
		Code:     "blog",
		Name:     "博客",
		IsSystem: 0,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newDeleteCtx(7)
	svc := NewApplicationSvc()
	if err := svc.Delete(ctx, &dtoapplication.ApplicationDeleteReq{AppID: entity.ID}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// 真实 dao 断言：软删除后按 ID 查不到
	got, err := dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil && got.ID != 0 {
		t.Fatalf("expected application soft-deleted, got %+v", got)
	}

	// 删除人应写入 7（对应原 stub 断言 deletedBy == 7）
	var deleted model.ApplicationEntity
	if err := db.Unscoped().Where("id = ?", entity.ID).First(&deleted).Error; err != nil {
		t.Fatalf("query deleted row: %v", err)
	}
	if deleted.DeletedBy != 7 {
		t.Fatalf("expected deletedBy 7, got %d", deleted.DeletedBy)
	}
}
