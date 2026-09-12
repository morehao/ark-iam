package svcapplication

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
)

// newDeleteCtx 构造带租户与操作人上下文的 gin.Context。
func newDeleteCtx(userID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, "1")
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

// TestDeleteBuiltInApplication：source=builtin（平台管理后台）禁删。
func TestDeleteBuiltInApplication(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})

	entity := &model.ApplicationEntity{
		Code:   "admin",
		Name:   "平台管理后台",
		Source: model.AppSourceBuiltin,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewApplicationSvc()
	err := svc.Delete(newDeleteCtx("0"), &dtoapplication.ApplicationDeleteReq{AppID: entity.ID})
	if err == nil {
		t.Fatal("expected error for built-in application")
	}
	if gerror.GetCode(err) != int(code.ApplicationBuiltInErr) {
		t.Fatalf("expected ApplicationBuiltInErr, got %v", err)
	}
}

// TestDeleteFirstPartyApplication：source=first_party（平台自建但非内置）可删——
// 删 is_system 后这类应用不得被误判为内置而拒绝删除。注意：种子的两个控制台应用都是 builtin，
// first_party 目前只可能来自运维自建（见 docs/design/system-design.md §4.3）。
func TestDeleteFirstPartyApplication(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})

	entity := &model.ApplicationEntity{
		Code:   "ops-app",
		Name:   "运维自建应用",
		Source: model.AppSourceFirstParty,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newDeleteCtx("7")
	svc := NewApplicationSvc()
	if err := svc.Delete(ctx, &dtoapplication.ApplicationDeleteReq{AppID: entity.ID}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

// TestDeleteThirdPartyApplication：source=third_party 可删，且删除人写入上下文用户。
func TestDeleteThirdPartyApplication(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationEntity{})

	entity := &model.ApplicationEntity{
		Code:   "blog",
		Name:   "博客",
		Source: model.AppSourceThirdParty,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newDeleteCtx("7")
	svc := NewApplicationSvc()
	if err := svc.Delete(ctx, &dtoapplication.ApplicationDeleteReq{AppID: entity.ID}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// 真实 dao 断言：软删除后按 ID 查不到
	got, err := dao.NewApplicationDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil && got.ID != "" {
		t.Fatalf("expected application soft-deleted, got %+v", got)
	}

	// 删除人应写入 7（对应原 stub 断言 deletedBy == 7）
	var deleted model.ApplicationEntity
	if err := db.Unscoped().Where("id = ?", entity.ID).First(&deleted).Error; err != nil {
		t.Fatalf("query deleted row: %v", err)
	}
	if deleted.DeletedBy != "7" {
		t.Fatalf("expected deletedBy 7, got %s", deleted.DeletedBy)
	}
}
