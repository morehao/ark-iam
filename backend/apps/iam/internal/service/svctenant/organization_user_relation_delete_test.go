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

func TestDeleteOrganizationUserRelationUsesTenantScopedCompositeLookup(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(71))
	ginCtx.Set(gcontext.KeyUserID, uint(9301))

	repo := &stubOrganizationUserRelationDeleteRepo{
		list: model.OrganizationUserRelationEntityList{{
			Model:          gorm.Model{ID: 109},
			TenantID:       71,
			OrganizationID: 201,
			UserID:         301,
		}},
	}
	installOrganizationUserRelationDeleteRepo(t, repo)

	svc := &organizationUserRelationSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationUserRelationDeleteReq{OrganizationID: 201, UserID: 301})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected lookup condition to be captured")
	}
	if repo.lastCond.TenantID != 71 || repo.lastCond.OrganizationID != 201 || repo.lastCond.UserID != 301 {
		t.Fatalf("unexpected composite lookup: %+v", repo.lastCond)
	}
	if repo.deletedID != 109 {
		t.Fatalf("expected delete by relation id 109, got %d", repo.deletedID)
	}
	if repo.deletedBy != 9301 {
		t.Fatalf("expected deletedBy 9301, got %d", repo.deletedBy)
	}
}

func TestDeleteOrganizationUserRelationReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(72))
	ginCtx.Set(gcontext.KeyUserID, uint(9302))

	repo := &stubOrganizationUserRelationDeleteRepo{}
	installOrganizationUserRelationDeleteRepo(t, repo)

	svc := &organizationUserRelationSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationUserRelationDeleteReq{OrganizationID: 201, UserID: 301})
	if err == nil {
		t.Fatalf("expected not exist error")
	}
	if repo.deletedID != 0 {
		t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
	}
}

type stubOrganizationUserRelationDeleteRepo struct {
	list      model.OrganizationUserRelationEntityList
	listErr   error
	deleteErr error
	lastCond  *dao.OrganizationUserRelationCond
	deletedID uint
	deletedBy uint
}

func (r *stubOrganizationUserRelationDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationUserRelationEntityList, error) {
	typed, _ := cond.(*dao.OrganizationUserRelationCond)
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

func (r *stubOrganizationUserRelationDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return r.deleteErr
}

func installOrganizationUserRelationDeleteRepo(t *testing.T, repo organizationUserRelationDeleteRepository) {
	t.Helper()
	prev := newOrganizationUserRelationDeleteRepo
	newOrganizationUserRelationDeleteRepo = func() organizationUserRelationDeleteRepository {
		return repo
	}
	t.Cleanup(func() {
		newOrganizationUserRelationDeleteRepo = prev
	})
}

var _ organizationUserRelationDeleteRepository = (*stubOrganizationUserRelationDeleteRepo)(nil)
