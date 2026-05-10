package svctenant

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

func TestDeleteOrganizationRoleUserRelationUsesTenantScopedCompositeLookup(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(81))
	ginCtx.Set(gcontext.KeyUserID, uint(9401))

	repo := &stubOrganizationRoleUserRelationDeleteRepo{
		list: model.OrganizationRoleUserRelationEntityList{{
			Model:              gorm.Model{ID: 119},
			TenantID:           81,
			OrganizationID:     401,
			OrganizationRoleID: 501,
			UserID:             601,
		}},
	}
	installOrganizationRoleUserRelationDeleteRepo(t, repo)

	svc := &organizationRoleUserRelationSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserRelationDeleteReq{OrganizationRoleID: 501, UserID: 601})
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

func TestDeleteOrganizationRoleUserRelationReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(82))
	ginCtx.Set(gcontext.KeyUserID, uint(9402))

	repo := &stubOrganizationRoleUserRelationDeleteRepo{}
	installOrganizationRoleUserRelationDeleteRepo(t, repo)

	svc := &organizationRoleUserRelationSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserRelationDeleteReq{OrganizationRoleID: 501, UserID: 601})
	if err == nil {
		t.Fatalf("expected not exist error")
	}
	if repo.deletedID != 0 {
		t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
	}
}

type stubOrganizationRoleUserRelationDeleteRepo struct {
	list      model.OrganizationRoleUserRelationEntityList
	listErr   error
	deleteErr error
	lastCond  *dao.OrganizationRoleUserRelationCond
	deletedID uint
	deletedBy uint
}

func (r *stubOrganizationRoleUserRelationDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationRoleUserRelationEntityList, error) {
	typed, _ := cond.(*dao.OrganizationRoleUserRelationCond)
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

func (r *stubOrganizationRoleUserRelationDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return r.deleteErr
}

func installOrganizationRoleUserRelationDeleteRepo(t *testing.T, repo organizationRoleUserRelationDeleteRepository) {
	t.Helper()
	prev := newOrganizationRoleUserRelationDeleteRepo
	newOrganizationRoleUserRelationDeleteRepo = func() organizationRoleUserRelationDeleteRepository {
		return repo
	}
	t.Cleanup(func() {
		newOrganizationRoleUserRelationDeleteRepo = prev
	})
}

var _ organizationRoleUserRelationDeleteRepository = (*stubOrganizationRoleUserRelationDeleteRepo)(nil)
