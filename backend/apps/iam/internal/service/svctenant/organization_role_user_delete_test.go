package svctenant

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func TestDeleteOrganizationRoleUserUsesTenantScopedCompositeLookup(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(81))
	ginCtx.Set(gcontext.KeyUserID, uint(9401))

	repo := &stubOrganizationRoleUserDeleteRepo{
		list: model.OrganizationRoleUserEntityList{{
			Model:              gorm.Model{ID: 119},
			TenantID:           81,
			OrganizationID:     401,
			OrganizationRoleID: 501,
			UserID:             601,
		}},
	}
	installOrganizationRoleUserDeleteRepo(t, repo)

	svc := &organizationRoleUserSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserDeleteReq{OrganizationRoleID: 501, UserID: 601})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected lookup condition to be captured")
	}
	if repo.lastCond.TenantID != 81 || repo.lastCond.OrganizationRoleID != 501 || repo.lastCond.UserID != 601 {
		t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
	}
	if repo.deletedID != 119 {
		t.Fatalf("expected delete by relation id 119, got %d", repo.deletedID)
	}
	if repo.deletedBy != 9401 {
		t.Fatalf("expected deletedBy 9401, got %d", repo.deletedBy)
	}
}

func TestDeleteOrganizationRoleUserReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(82))
	ginCtx.Set(gcontext.KeyUserID, uint(9402))

	repo := &stubOrganizationRoleUserDeleteRepo{}
	installOrganizationRoleUserDeleteRepo(t, repo)

	svc := &organizationRoleUserSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserDeleteReq{OrganizationRoleID: 501, UserID: 601})
	if err == nil {
		t.Fatalf("expected not exist error")
	}
	if repo.deletedID != 0 {
		t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
	}
}

type stubOrganizationRoleUserDeleteRepo struct {
	list      model.OrganizationRoleUserEntityList
	listErr   error
	deleteErr error
	lastCond  *dao.OrganizationRoleUserCond
	deletedID uint
	deletedBy uint
}

func (r *stubOrganizationRoleUserDeleteRepo) GetListByCond(ctx context.Context, cond gormdao.Cond) (model.OrganizationRoleUserEntityList, error) {
	typed, _ := cond.(*dao.OrganizationRoleUserCond)
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

func (r *stubOrganizationRoleUserDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return r.deleteErr
}

func installOrganizationRoleUserDeleteRepo(t *testing.T, repo organizationRoleUserDeleteRepository) {
	t.Helper()
	prev := newOrganizationRoleUserDeleteRepo
	newOrganizationRoleUserDeleteRepo = func() organizationRoleUserDeleteRepository {
		return repo
	}
	t.Cleanup(func() {
		newOrganizationRoleUserDeleteRepo = prev
	})
}

var _ organizationRoleUserDeleteRepository = (*stubOrganizationRoleUserDeleteRepo)(nil)
