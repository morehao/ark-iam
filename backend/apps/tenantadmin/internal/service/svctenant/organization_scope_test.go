package svctenant

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

func TestOrganizationDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(41))

	repo := &stubOrganizationScopeRepo{
		detail: &model.OrganizationEntity{Model: gorm.Model{ID: 8}, TenantID: 99},
	}
	installOrganizationScopeRepo(t, repo)

	svc := &organizationSvc{}
	resp, err := svc.Detail(ginCtx, &dtotenant.OrganizationDetailReq{OrganizationID: 8})
	if err == nil {
		t.Fatalf("expected cross-tenant organization detail to fail, resp=%+v", resp)
	}
}

func TestOrganizationRoleDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(51))

	repo := &stubOrganizationRoleScopeRepo{
		detail: &model.OrganizationRoleEntity{Model: gorm.Model{ID: 9}, TenantID: 98},
	}
	installOrganizationRoleScopeRepo(t, repo)

	svc := &organizationRoleSvc{}
	resp, err := svc.Detail(ginCtx, &dtotenant.OrganizationRoleDetailReq{OrganizationRoleID: 9})
	if err == nil {
		t.Fatalf("expected cross-tenant organization role detail to fail, resp=%+v", resp)
	}
}

func TestOrganizationRoleCreateRejectsCrossTenantOrganization(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(52))
	ginCtx.Set(gcontext.KeyUserID, uint(1001))

	repo := &stubOrganizationRoleScopeRepo{
		organization: &model.OrganizationEntity{Model: gorm.Model{ID: 6}, TenantID: 77},
	}
	installOrganizationRoleScopeRepo(t, repo)

	svc := &organizationRoleSvc{}
	resp, err := svc.Create(ginCtx, &dtotenant.OrganizationRoleCreateReq{TenantID: 52, OrganizationID: 6, Name: "ops"})
	if err == nil {
		t.Fatalf("expected cross-tenant organization create to fail, resp=%+v", resp)
	}
	if repo.inserted != nil {
		t.Fatalf("expected no insert when organization is cross-tenant")
	}
}

type stubOrganizationScopeRepo struct {
	detail *model.OrganizationEntity
	err    error
}

func (r *stubOrganizationScopeRepo) GetByID(ctx context.Context, id uint) (*model.OrganizationEntity, error) {
	return r.detail, r.err
}

func installOrganizationScopeRepo(t *testing.T, repo organizationScopeRepository) {
	t.Helper()
	prev := newOrganizationScopeRepo
	newOrganizationScopeRepo = func() organizationScopeRepository {
		return repo
	}
	t.Cleanup(func() {
		newOrganizationScopeRepo = prev
	})
}

type stubOrganizationRoleScopeRepo struct {
	detail       *model.OrganizationRoleEntity
	organization *model.OrganizationEntity
	err          error
	inserted     *model.OrganizationRoleEntity
}

func (r *stubOrganizationRoleScopeRepo) GetByID(ctx context.Context, id uint) (*model.OrganizationRoleEntity, error) {
	return r.detail, r.err
}

func (r *stubOrganizationRoleScopeRepo) GetOrganizationByID(ctx context.Context, id uint) (*model.OrganizationEntity, error) {
	return r.organization, r.err
}

func (r *stubOrganizationRoleScopeRepo) Insert(ctx context.Context, entity *model.OrganizationRoleEntity) error {
	clone := *entity
	r.inserted = &clone
	return r.err
}

func installOrganizationRoleScopeRepo(t *testing.T, repo organizationRoleScopeRepository) {
	t.Helper()
	prev := newOrganizationRoleScopeRepo
	newOrganizationRoleScopeRepo = func() organizationRoleScopeRepository {
		return repo
	}
	t.Cleanup(func() {
		newOrganizationRoleScopeRepo = prev
	})
}

var _ organizationScopeRepository = (*stubOrganizationScopeRepo)(nil)
var _ organizationRoleScopeRepository = (*stubOrganizationRoleScopeRepo)(nil)
