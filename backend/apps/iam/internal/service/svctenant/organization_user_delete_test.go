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

func TestDeleteOrganizationUserUsesTenantScopedCompositeLookup(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(71))
	ginCtx.Set(gcontext.KeyUserID, uint(9301))

	repo := &stubOrganizationUserDeleteRepo{
		list: model.OrganizationUserEntityList{{
			Model:          gorm.Model{ID: 109},
			TenantID:       71,
			OrganizationID: 201,
			UserID:         301,
		}},
	}
	installOrganizationUserDeleteRepo(t, repo)

	svc := &organizationUserSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationUserDeleteReq{OrganizationID: 201, UserID: 301})
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

func TestDeleteOrganizationUserReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(72))
	ginCtx.Set(gcontext.KeyUserID, uint(9302))

	repo := &stubOrganizationUserDeleteRepo{}
	installOrganizationUserDeleteRepo(t, repo)

	svc := &organizationUserSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationUserDeleteReq{OrganizationID: 201, UserID: 301})
	if err == nil {
		t.Fatalf("expected not exist error")
	}
	if repo.deletedID != 0 {
		t.Fatalf("expected no delete call, got deletedID=%d", repo.deletedID)
	}
}

type stubOrganizationUserDeleteRepo struct {
	list      model.OrganizationUserEntityList
	listErr   error
	deleteErr error
	lastCond  *dao.OrganizationUserCond
	deletedID uint
	deletedBy uint
}

func (r *stubOrganizationUserDeleteRepo) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.OrganizationUserEntityList, error) {
	typed, _ := cond.(*dao.OrganizationUserCond)
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

func (r *stubOrganizationUserDeleteRepo) Delete(ctx context.Context, id uint, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return r.deleteErr
}

func installOrganizationUserDeleteRepo(t *testing.T, repo organizationUserDeleteRepository) {
	t.Helper()
	prev := newOrganizationUserDeleteRepo
	newOrganizationUserDeleteRepo = func() organizationUserDeleteRepository {
		return repo
	}
	t.Cleanup(func() {
		newOrganizationUserDeleteRepo = prev
	})
}

var _ organizationUserDeleteRepository = (*stubOrganizationUserDeleteRepo)(nil)
