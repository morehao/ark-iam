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

func TestDeleteRoleMenuUsesTenantScopedCompositeLookup(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(51))
	ginCtx.Set(gcontext.KeyUserID, uint(9101))

	repo := &stubRoleMenuDeleteRepo{
		list: model.RoleMenuEntityList{{
			Model:    gorm.Model{ID: 88},
			TenantID: 51,
			RoleID:   21,
			MenuID:   43,
		}},
	}
	installRoleMenuDeleteRepo(t, repo)

	svc := &roleMenuSvc{}
	err := svc.Delete(ginCtx, &dtopermission.RoleMenuDeleteReq{TenantID: 999, RoleID: 21, MenuID: 43})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected lookup condition to be captured")
	}
	if repo.lastCond.TenantID != 51 || repo.lastCond.RoleID != 21 || repo.lastCond.MenuID != 43 {
		t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
	}
	if repo.deletedID != 88 {
		t.Fatalf("expected delete by relation id 88, got %d", repo.deletedID)
	}
	if repo.deletedBy != 9101 {
		t.Fatalf("expected deletedBy 9101, got %d", repo.deletedBy)
	}
}

func TestDeleteRoleMenuReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(52))
	ginCtx.Set(gcontext.KeyUserID, uint(9102))

	repo := &stubRoleMenuDeleteRepo{}
	installRoleMenuDeleteRepo(t, repo)

	svc := &roleMenuSvc{}
	err := svc.Delete(ginCtx, &dtopermission.RoleMenuDeleteReq{RoleID: 21, MenuID: 43})
	if err == nil {
		t.Fatalf("expected not exist error")
	}
	if repo.deletedID != 0 {
		t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
	}
}

type stubRoleMenuDeleteRepo struct {
	list      model.RoleMenuEntityList
	listErr   error
	deleteErr error
	lastCond  *dao.RoleMenuCond
	deletedID uint
	deletedBy uint
}

func (r *stubRoleMenuDeleteRepo) GetListByCond(ctx context.Context, cond gormdao.Cond) (model.RoleMenuEntityList, error) {
	typed, _ := cond.(*dao.RoleMenuCond)
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

func (r *stubRoleMenuDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return r.deleteErr
}

func installRoleMenuDeleteRepo(t *testing.T, repo roleMenuDeleteRepository) {
	t.Helper()
	prev := newRoleMenuDeleteRepo
	newRoleMenuDeleteRepo = func() roleMenuDeleteRepository {
		return repo
	}
	t.Cleanup(func() {
		newRoleMenuDeleteRepo = prev
	})
}

var _ roleMenuDeleteRepository = (*stubRoleMenuDeleteRepo)(nil)
