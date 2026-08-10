package svcpermission

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func TestMenuDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(21))

	repo := &stubMenuScopeRepo{detail: &model.MenuEntity{Model: gorm.Model{ID: 5}}}
	installMenuScopeRepo(t, repo)

	svc := &menuSvc{}
	_, err := svc.Detail(ginCtx, &dtopermission.MenuDetailReq{MenuID: 5})
	if err != nil {
		t.Fatalf("unexpected error for menu detail: %v", err)
	}
}

func TestMenuPageListUsesAppID(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(22))

	repo := &stubMenuScopeRepo{}
	installMenuScopeRepo(t, repo)

	svc := &menuSvc{}
	_, _ = svc.PageList(ginCtx, &dtopermission.MenuPageListReq{AppID: 10})
	if repo.lastCond == nil || repo.lastCond.AppID != 10 {
		t.Fatalf("expected application 10, got %+v", repo.lastCond)
	}
}

func TestMenuTreeUsesAppID(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(23))

	repo := &stubMenuScopeRepo{recordTree: true}
	installMenuScopeRepo(t, repo)

	svc := &menuSvc{}
	_, _ = svc.Tree(ginCtx, &dtopermission.MenuTreeReq{AppID: 10})
	if repo.lastTreeCond == nil || repo.lastTreeCond.AppID != 10 {
		t.Fatalf("expected application 10, got %+v", repo.lastTreeCond)
	}
}

func TestRoleDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(31))

	repo := &stubRoleScopeRepo{detail: &model.RoleEntity{Model: gorm.Model{ID: 6}, TenantID: 77}}
	installRoleScopeRepo(t, repo)

	svc := &roleSvc{}
	_, err := svc.Detail(ginCtx, &dtopermission.RoleDetailReq{RoleID: 6})
	if err == nil {
		t.Fatalf("expected cross-tenant role detail to fail")
	}
}

func TestRolePageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(32))

	repo := &stubRoleScopeRepo{}
	installRoleScopeRepo(t, repo)

	svc := &roleSvc{}
	_, _ = svc.PageList(ginCtx, &dtopermission.RolePageListReq{TenantID: 99})
	if repo.lastCond == nil || repo.lastCond.TenantID != 32 {
		t.Fatalf("expected tenant 32 from context, got %+v", repo.lastCond)
	}
}

func TestUserRoleCreateRejectsCrossTenantRole(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(33))
	ginCtx.Set(gcontext.KeyUserID, uint(1001))

	repo := &stubUserRoleCreateScopeRepo{role: &model.RoleEntity{Model: gorm.Model{ID: 9}, TenantID: 44}}
	installUserRoleCreateScopeRepo(t, repo)

	svc := &userRoleSvc{}
	_, err := svc.Create(ginCtx, &dtopermission.UserRoleCreateReq{TenantID: 33, UserID: 1, RoleID: 9})
	if err == nil {
		t.Fatalf("expected cross-tenant role reference to fail")
	}
	if repo.inserted != nil {
		t.Fatalf("expected no insert for cross-tenant role")
	}
}

func TestResourceDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(41))

	repo := &stubResourceScopeRepo{detail: &model.ResourceEntity{Model: gorm.Model{ID: 7}, TenantID: 66}}
	installResourceScopeRepo(t, repo)

	svc := &resourceSvc{}
	_, err := svc.Detail(ginCtx, &dtopermission.ResourceDetailReq{ResourceID: 7})
	if err == nil {
		t.Fatalf("expected cross-tenant resource detail to fail")
	}
}

func TestResourcePageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(42))

	repo := &stubResourceScopeRepo{}
	installResourceScopeRepo(t, repo)

	svc := &resourceSvc{}
	_, _ = svc.PageList(ginCtx, &dtopermission.ResourcePageListReq{TenantID: 99})
	if repo.lastCond == nil || repo.lastCond.TenantID != 42 {
		t.Fatalf("expected tenant 42 from context, got %+v", repo.lastCond)
	}
}

func TestScopeDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(51))

	repo := &stubScopeScopeRepo{detail: &model.ScopeEntity{Model: gorm.Model{ID: 8}, TenantID: 65}}
	installScopeScopeRepo(t, repo)

	svc := &scopeSvc{}
	_, err := svc.Detail(ginCtx, &dtopermission.ScopeDetailReq{ScopeID: 8})
	if err == nil {
		t.Fatalf("expected cross-tenant scope detail to fail")
	}
}

func TestScopePageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(52))

	repo := &stubScopeScopeRepo{}
	installScopeScopeRepo(t, repo)

	svc := &scopeSvc{}
	_, _ = svc.PageList(ginCtx, &dtopermission.ScopePageListReq{TenantID: 99})
	if repo.lastCond == nil || repo.lastCond.TenantID != 52 {
		t.Fatalf("expected tenant 52 from context, got %+v", repo.lastCond)
	}
}

func TestScopeCreateRejectsCrossTenantResource(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(53))
	ginCtx.Set(gcontext.KeyUserID, uint(1002))

	repo := &stubScopeScopeRepo{resource: &model.ResourceEntity{Model: gorm.Model{ID: 10}, TenantID: 64}}
	installScopeScopeRepo(t, repo)

	svc := &scopeSvc{}
	_, err := svc.Create(ginCtx, &dtopermission.ScopeCreateReq{TenantID: 53, ResourceID: 10, Name: "read"})
	if err == nil {
		t.Fatalf("expected cross-tenant resource reference to fail")
	}
	if repo.inserted != nil {
		t.Fatalf("expected no insert for cross-tenant resource")
	}
}

type stubMenuScopeRepo struct {
	detail       *model.MenuEntity
	pageList     model.MenuEntityList
	total        int64
	err          error
	lastCond     *dao.MenuCond
	lastTreeCond *dao.MenuCond
	recordTree   bool
}

func (r *stubMenuScopeRepo) GetByID(ctx context.Context, id uint) (*model.MenuEntity, error) {
	return r.detail, r.err
}
func (r *stubMenuScopeRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.MenuEntityList, int64, error) {
	typed, _ := cond.(*dao.MenuCond)
	clone := *typed
	if typed.BaseCond != nil {
		base := *typed.BaseCond
		clone.BaseCond = &base
	}
	if r.recordTree {
		r.lastTreeCond = &clone
	} else {
		r.lastCond = &clone
	}
	return r.pageList, r.total, r.err
}

type stubRoleScopeRepo struct {
	detail   *model.RoleEntity
	pageList model.RoleEntityList
	total    int64
	err      error
	lastCond *dao.RoleCond
}

func (r *stubRoleScopeRepo) GetByID(ctx context.Context, id uint) (*model.RoleEntity, error) {
	return r.detail, r.err
}
func (r *stubRoleScopeRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.RoleEntityList, int64, error) {
	typed, _ := cond.(*dao.RoleCond)
	clone := *typed
	if typed.BaseCond != nil {
		base := *typed.BaseCond
		clone.BaseCond = &base
	}
	r.lastCond = &clone
	return r.pageList, r.total, r.err
}

type stubUserRoleCreateScopeRepo struct {
	role     *model.RoleEntity
	err      error
	inserted *model.UserRoleEntity
}

func (r *stubUserRoleCreateScopeRepo) GetRoleByID(ctx context.Context, id uint) (*model.RoleEntity, error) {
	return r.role, r.err
}
func (r *stubUserRoleCreateScopeRepo) Insert(ctx context.Context, entity *model.UserRoleEntity) error {
	clone := *entity
	r.inserted = &clone
	return r.err
}

type stubResourceScopeRepo struct {
	detail   *model.ResourceEntity
	pageList model.ResourceEntityList
	total    int64
	err      error
	lastCond *dao.ResourceCond
}

func (r *stubResourceScopeRepo) GetByID(ctx context.Context, id uint) (*model.ResourceEntity, error) {
	return r.detail, r.err
}
func (r *stubResourceScopeRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.ResourceEntityList, int64, error) {
	typed, _ := cond.(*dao.ResourceCond)
	clone := *typed
	if typed.BaseCond != nil {
		base := *typed.BaseCond
		clone.BaseCond = &base
	}
	r.lastCond = &clone
	return r.pageList, r.total, r.err
}

type stubScopeScopeRepo struct {
	detail   *model.ScopeEntity
	resource *model.ResourceEntity
	pageList model.ScopeEntityList
	total    int64
	err      error
	lastCond *dao.ScopeCond
	inserted *model.ScopeEntity
}

func (r *stubScopeScopeRepo) GetByID(ctx context.Context, id uint) (*model.ScopeEntity, error) {
	return r.detail, r.err
}
func (r *stubScopeScopeRepo) GetResourceByID(ctx context.Context, id uint) (*model.ResourceEntity, error) {
	return r.resource, r.err
}
func (r *stubScopeScopeRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.ScopeEntityList, int64, error) {
	typed, _ := cond.(*dao.ScopeCond)
	clone := *typed
	if typed.BaseCond != nil {
		base := *typed.BaseCond
		clone.BaseCond = &base
	}
	r.lastCond = &clone
	return r.pageList, r.total, r.err
}
func (r *stubScopeScopeRepo) Insert(ctx context.Context, entity *model.ScopeEntity) error {
	clone := *entity
	r.inserted = &clone
	return r.err
}

func installMenuScopeRepo(t *testing.T, repo menuScopeRepository) {
	t.Helper()
	prev := newMenuScopeRepo
	newMenuScopeRepo = func() menuScopeRepository { return repo }
	t.Cleanup(func() { newMenuScopeRepo = prev })
}

func installRoleScopeRepo(t *testing.T, repo roleScopeRepository) {
	t.Helper()
	prev := newRoleScopeRepo
	newRoleScopeRepo = func() roleScopeRepository { return repo }
	t.Cleanup(func() { newRoleScopeRepo = prev })
}

func installUserRoleCreateScopeRepo(t *testing.T, repo userRoleCreateScopeRepository) {
	t.Helper()
	prev := newUserRoleCreateScopeRepo
	newUserRoleCreateScopeRepo = func() userRoleCreateScopeRepository { return repo }
	t.Cleanup(func() { newUserRoleCreateScopeRepo = prev })
}

func installResourceScopeRepo(t *testing.T, repo resourceScopeRepository) {
	t.Helper()
	prev := newResourceScopeRepo
	newResourceScopeRepo = func() resourceScopeRepository { return repo }
	t.Cleanup(func() { newResourceScopeRepo = prev })
}

func installScopeScopeRepo(t *testing.T, repo scopeScopeRepository) {
	t.Helper()
	prev := newScopeScopeRepo
	newScopeScopeRepo = func() scopeScopeRepository { return repo }
	t.Cleanup(func() { newScopeScopeRepo = prev })
}

var _ menuScopeRepository = (*stubMenuScopeRepo)(nil)
var _ roleScopeRepository = (*stubRoleScopeRepo)(nil)
var _ userRoleCreateScopeRepository = (*stubUserRoleCreateScopeRepo)(nil)
var _ resourceScopeRepository = (*stubResourceScopeRepo)(nil)
var _ scopeScopeRepository = (*stubScopeScopeRepo)(nil)
