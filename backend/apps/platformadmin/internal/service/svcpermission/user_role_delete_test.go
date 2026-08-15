package svcpermission

import (
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/testutil"
)

func TestDeleteUserRoleUsesTenantScopedCompositeLookup(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserRoleEntity{})
	ctx := newGinCtx("41", "9001")

	relation := &model.UserRoleEntity{TenantID: "41", UserID: "12", RoleID: "34"}
	if err := db.Create(relation).Error; err != nil {
		t.Fatalf("seed user role: %v", err)
	}

	svc := &userRoleSvc{}
	err := svc.Delete(ctx, &dtopermission.UserRoleDeleteReq{TenantID: "999", UserID: "12", RoleID: "34"})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// 软删除后按租户+用户+角色组合条件应查不到
	left, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{
		TenantID: "41",
		UserID:   "12",
		RoleID:   "34",
	})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected relation deleted, got %+v", left)
	}

	var softDeleted model.UserRoleEntity
	if err := db.Unscoped().Where("id = ?", relation.ID).First(&softDeleted).Error; err != nil {
		t.Fatalf("query soft-deleted row: %v", err)
	}
	if softDeleted.DeletedAt.Time.IsZero() {
		t.Fatalf("expected soft delete timestamp")
	}
	if softDeleted.DeletedBy != "9001" {
		t.Fatalf("expected deletedBy 9001, got %s", softDeleted.DeletedBy)
	}
}

func TestDeleteUserRoleReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	testutil.SetupSQLite(t, &model.UserRoleEntity{})
	ctx := newGinCtx("42", "9002")

	svc := &userRoleSvc{}
	err := svc.Delete(ctx, &dtopermission.UserRoleDeleteReq{UserID: "12", RoleID: "34"})
	if err == nil {
		t.Fatalf("expected not exist error")
	}

	left, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{
		TenantID: "42",
		UserID:   "12",
		RoleID:   "34",
	})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected no delete call, got %+v", left)
	}
}
