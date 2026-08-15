package svcpermission

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

func newGinCtx(tenantID, userID uint) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

func TestDeleteRoleMenuUsesTenantScopedCompositeLookup(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleMenuEntity{})
	ctx := newGinCtx(51, 9101)

	relation := &model.RoleMenuEntity{TenantID: 51, RoleID: 21, MenuID: 43}
	if err := db.Create(relation).Error; err != nil {
		t.Fatalf("seed role menu: %v", err)
	}

	svc := &roleMenuSvc{}
	err := svc.Delete(ctx, &dtopermission.RoleMenuDeleteReq{TenantID: 999, RoleID: 21, MenuID: 43})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// 软删除后按租户+角色+菜单组合条件应查不到
	left, err := dao.NewRoleMenuDao().GetListByCond(ctx, &dao.RoleMenuCond{
		TenantID: 51,
		RoleID:   21,
		MenuID:   43,
	})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected relation deleted, got %+v", left)
	}

	// 已删除记录应记录删除人与删除时间（软删除）
	var softDeleted model.RoleMenuEntity
	if err := db.Unscoped().Where("id = ?", relation.ID).First(&softDeleted).Error; err != nil {
		t.Fatalf("query soft-deleted row: %v", err)
	}
	if softDeleted.DeletedAt.Time.IsZero() {
		t.Fatalf("expected soft delete timestamp")
	}
	if softDeleted.DeletedBy != 9101 {
		t.Fatalf("expected deletedBy 9101, got %d", softDeleted.DeletedBy)
	}
}

func TestDeleteRoleMenuReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	testutil.SetupSQLite(t, &model.RoleMenuEntity{})
	ctx := newGinCtx(52, 9102)

	svc := &roleMenuSvc{}
	err := svc.Delete(ctx, &dtopermission.RoleMenuDeleteReq{RoleID: 21, MenuID: 43})
	if err == nil {
		t.Fatalf("expected not exist error")
	}

	left, err := dao.NewRoleMenuDao().GetListByCond(ctx, &dao.RoleMenuCond{
		TenantID: 52,
		RoleID:   21,
		MenuID:   43,
	})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected no delete call, got %+v", left)
	}
}
