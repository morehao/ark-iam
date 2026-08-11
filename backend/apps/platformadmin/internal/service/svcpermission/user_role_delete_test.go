package svcpermission

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func TestDeleteUserRoleUsesTenantScopedCompositeLookup(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(41))
	ginCtx.Set(gcontext.KeyUserID, uint(9001))

	repo := &stubUserRoleDeleteRepo{
		list: model.UserRoleEntityList{{
			Model:    gorm.Model{ID: 77},
			TenantID: 41,
			UserID:   12,
			RoleID:   34,
		}},
	}
	installUserRoleDeleteRepo(t, repo)

	svc := &userRoleSvc{}
	err := svc.Delete(ginCtx, &dtopermission.UserRoleDeleteReq{TenantID: 999, UserID: 12, RoleID: 34})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected delete lookup condition to be captured")
	}
	if repo.lastCond.TenantID != 41 {
		t.Fatalf("expected tenant lookup 41 from context, got %d", repo.lastCond.TenantID)
	}
	if repo.lastCond.UserID != 12 || repo.lastCond.RoleID != 34 {
		t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
	}
	if repo.deletedID != 77 {
		t.Fatalf("expected delete by relation id 77, got %d", repo.deletedID)
	}
	if repo.deletedBy != 9001 {
		t.Fatalf("expected deletedBy 9001, got %d", repo.deletedBy)
	}
}

func TestDeleteUserRoleReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(42))
	ginCtx.Set(gcontext.KeyUserID, uint(9002))

	repo := &stubUserRoleDeleteRepo{}
	installUserRoleDeleteRepo(t, repo)

	svc := &userRoleSvc{}
	err := svc.Delete(ginCtx, &dtopermission.UserRoleDeleteReq{UserID: 12, RoleID: 34})
	if err == nil {
		t.Fatalf("expected not exist error")
	}
	if repo.deletedID != 0 {
		t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
	}
}

type stubUserRoleDeleteRepo struct {
	list      model.UserRoleEntityList
	listErr   error
	deleteErr error
	lastCond  *dao.UserRoleCond
	deletedID uint
	deletedBy uint
}

func (r *stubUserRoleDeleteRepo) GetListByCond(ctx context.Context, cond gormdao.Cond) (model.UserRoleEntityList, error) {
	typed, _ := cond.(*dao.UserRoleCond)
	if typed != nil {
		clone := *typed
		if typed.BaseCond != nil {
			base := *typed.BaseCond
			clone.BaseCond = &base
		}
		r.lastCond = &clone
	}
	return r.list, r.listErr
}

func (r *stubUserRoleDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return r.deleteErr
}

func installUserRoleDeleteRepo(t *testing.T, repo userRoleDeleteRepository) {
	t.Helper()
	prev := newUserRoleDeleteRepo
	newUserRoleDeleteRepo = func() userRoleDeleteRepository {
		return repo
	}
	t.Cleanup(func() {
		newUserRoleDeleteRepo = prev
	})
}

var _ userRoleDeleteRepository = (*stubUserRoleDeleteRepo)(nil)
