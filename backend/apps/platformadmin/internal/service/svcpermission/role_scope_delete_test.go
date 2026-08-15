package svcpermission

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
)

func TestDeleteRoleScopeUsesTenantScopedCompositeLookup(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.RoleScopeEntity{})
	ctx := newGinCtx(61, 9201)

	relation := &model.RoleScopeEntity{TenantID: 61, RoleID: 31, ScopeID: 53}
	if err := db.Create(relation).Error; err != nil {
		t.Fatalf("seed role scope: %v", err)
	}

	svc := &roleScopeSvc{}
	err := svc.Delete(ctx, &dtopermission.RoleScopeDeleteReq{TenantID: 999, RoleID: 31, ScopeID: 53})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// 软删除后按租户+角色+权限组合条件应查不到
	left, err := dao.NewRoleScopeDao().GetListByCond(ctx, &dao.RoleScopeCond{
		TenantID: 61,
		RoleID:   31,
		ScopeID:  53,
	})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected relation deleted, got %+v", left)
	}

	var softDeleted model.RoleScopeEntity
	if err := db.Unscoped().Where("id = ?", relation.ID).First(&softDeleted).Error; err != nil {
		t.Fatalf("query soft-deleted row: %v", err)
	}
	if softDeleted.DeletedAt.Time.IsZero() {
		t.Fatalf("expected soft delete timestamp")
	}
	if softDeleted.DeletedBy != 9201 {
		t.Fatalf("expected deletedBy 9201, got %d", softDeleted.DeletedBy)
	}
}

func TestDeleteRoleScopeReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	testutil.SetupSQLite(t, &model.RoleScopeEntity{})
	ctx := newGinCtx(62, 9202)

	svc := &roleScopeSvc{}
	err := svc.Delete(ctx, &dtopermission.RoleScopeDeleteReq{RoleID: 31, ScopeID: 53})
	if err == nil {
		t.Fatalf("expected not exist error")
	}

	left, err := dao.NewRoleScopeDao().GetListByCond(ctx, &dao.RoleScopeCond{
		TenantID: 62,
		RoleID:   31,
		ScopeID:  53,
	})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected no delete call, got %+v", left)
	}
}
