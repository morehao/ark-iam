package svcpermission

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

func TestDeleteRoleScopeUsesTenantScopedCompositeLookup(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(61))
	ginCtx.Set(gcontext.KeyUserID, uint(9201))

	repo := &stubRoleScopeDeleteRepo{
		list: model.RoleScopeEntityList{{
			Model:    gorm.Model{ID: 99},
			TenantID: 61,
			RoleID:   31,
			ScopeID:  53,
		}},
	}
	installRoleScopeDeleteRepo(t, repo)

	svc := &roleScopeSvc{}
	err := svc.Delete(ginCtx, &dtopermission.RoleScopeDeleteReq{TenantID: 999, RoleID: 31, ScopeID: 53})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected lookup condition to be captured")
	}
	if repo.lastCond.TenantID != 61 || repo.lastCond.RoleID != 31 || repo.lastCond.ScopeID != 53 {
		t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
	}
	if repo.deletedID != 99 {
		t.Fatalf("expected delete by relation id 99, got %d", repo.deletedID)
	}
	if repo.deletedBy != 9201 {
		t.Fatalf("expected deletedBy 9201, got %d", repo.deletedBy)
	}
}

func TestDeleteRoleScopeReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(62))
	ginCtx.Set(gcontext.KeyUserID, uint(9202))

	repo := &stubRoleScopeDeleteRepo{}
	installRoleScopeDeleteRepo(t, repo)

	svc := &roleScopeSvc{}
	err := svc.Delete(ginCtx, &dtopermission.RoleScopeDeleteReq{RoleID: 31, ScopeID: 53})
	if err == nil {
		t.Fatalf("expected not exist error")
	}
	if repo.deletedID != 0 {
		t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
	}
}

type stubRoleScopeDeleteRepo struct {
	list      model.RoleScopeEntityList
	listErr   error
	deleteErr error
	lastCond  *dao.RoleScopeCond
	deletedID uint
	deletedBy uint
}

func (r *stubRoleScopeDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.RoleScopeEntityList, error) {
	typed, _ := cond.(*dao.RoleScopeCond)
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

func (r *stubRoleScopeDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return r.deleteErr
}

func installRoleScopeDeleteRepo(t *testing.T, repo roleScopeDeleteRepository) {
	t.Helper()
	prev := newRoleScopeDeleteRepo
	newRoleScopeDeleteRepo = func() roleScopeDeleteRepository {
		return repo
	}
	t.Cleanup(func() {
		newRoleScopeDeleteRepo = prev
	})
}

var _ roleScopeDeleteRepository = (*stubRoleScopeDeleteRepo)(nil)
