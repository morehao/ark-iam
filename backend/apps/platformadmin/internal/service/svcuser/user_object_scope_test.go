package svcuser

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

func TestUserDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(81))

	repo := &stubUserObjectScopeRepo{detail: &model.UserEntity{Model: gorm.Model{ID: 7}, TenantID: 99, Profile: []byte("{}"), CustomData: []byte("{}")}}
	installUserObjectScopeRepo(t, repo)

	svc := &userSvc{}
	_, err := svc.Detail(ginCtx, &dtouser.UserDetailReq{UserID: 7})
	if err == nil {
		t.Fatalf("expected cross-tenant user detail to fail")
	}
}

func TestUserPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(82))

	repo := &stubUserObjectScopeRepo{}
	installUserObjectScopeRepo(t, repo)

	svc := &userSvc{}
	_, _ = svc.PageList(ginCtx, &dtouser.UserPageListReq{TenantID: 99})
	if repo.lastUserCond == nil || repo.lastUserCond.TenantID != 82 {
		t.Fatalf("expected tenant 82 from context, got %+v", repo.lastUserCond)
	}
}

func TestDetailUserLoginLogRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(83))

	repo := &stubUserLoginLogDetailRepo{detail: &model.UserLoginLogEntity{Model: gorm.Model{ID: 8, CreatedAt: time.Unix(1700000000, 0)}, TenantID: 91}}
	installUserLoginLogDetailRepo(t, repo)

	svc := &userSvc{}
	_, err := svc.DetailUserLoginLog(ginCtx, &dtouser.UserLoginLogDetailReq{UserLoginLogID: 8})
	if err == nil {
		t.Fatalf("expected cross-tenant login log detail to fail")
	}
}

func TestUserIdentityDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(84))

	repo := &stubUserIdentityScopeRepo{detail: &model.UserIdentityEntity{Model: gorm.Model{ID: 9}, PersonID: 92, Detail: []byte("{}")}}
	installUserIdentityScopeRepo(t, repo)
	installUserIdentityUserResolver(t, &stubUserIdentityUserRepo{detail: &model.UserEntity{Model: gorm.Model{ID: 9}, TenantID: 99, PersonID: 92}})

	svc := &userIdentitySvc{}
	_, err := svc.Detail(ginCtx, &dtouser.UserIdentityDetailReq{UserIdentityID: 9})
	if err == nil {
		t.Fatalf("expected cross-tenant identity detail to fail")
	}
}

func TestUserIdentityPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(85))

	repo := &stubUserIdentityScopeRepo{}
	installUserIdentityScopeRepo(t, repo)

	svc := &userIdentitySvc{}
	_, _ = svc.PageList(ginCtx, &dtouser.UserIdentityPageListReq{UserID: 99})
	if repo.lastCond == nil || repo.lastCond.PersonID != 99 {
		t.Fatalf("expected PersonID 99 from request, got %+v", repo.lastCond)
	}
}

type stubUserObjectScopeRepo struct {
	detail       *model.UserEntity
	pageList     model.UserEntityList
	total        int64
	err          error
	lastUserCond *dao.UserCond
}

func (r *stubUserObjectScopeRepo) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	return r.detail, r.err
}

func (r *stubUserObjectScopeRepo) GetPageListByCond(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, int64, error) {
	clone := *cond
	if cond.BaseCond != nil {
		base := *cond.BaseCond
		clone.BaseCond = &base
	}
	r.lastUserCond = &clone
	return r.pageList, r.total, r.err
}

func installUserObjectScopeRepo(t *testing.T, repo userObjectScopeRepository) {
	t.Helper()
	prevUser := newUserObjectScopeRepo
	prevQuery := newUserQueryRepo
	newUserObjectScopeRepo = func() userObjectScopeRepository { return repo }
	newUserQueryRepo = func() userQueryRepository { return repo }
	t.Cleanup(func() {
		newUserObjectScopeRepo = prevUser
		newUserQueryRepo = prevQuery
	})
}

type stubUserLoginLogDetailRepo struct {
	detail *model.UserLoginLogEntity
	err    error
}

func (r *stubUserLoginLogDetailRepo) GetByID(ctx context.Context, id uint) (*model.UserLoginLogEntity, error) {
	return r.detail, r.err
}

func installUserLoginLogDetailRepo(t *testing.T, repo userLoginLogDetailRepository) {
	t.Helper()
	prev := newUserLoginLogDetailRepo
	newUserLoginLogDetailRepo = func() userLoginLogDetailRepository { return repo }
	t.Cleanup(func() { newUserLoginLogDetailRepo = prev })
}

type stubUserIdentityScopeRepo struct {
	detail   *model.UserIdentityEntity
	pageList model.UserIdentityEntityList
	total    int64
	err      error
	lastCond *dao.UserIdentityCond
}

func (r *stubUserIdentityScopeRepo) GetByID(ctx context.Context, id uint) (*model.UserIdentityEntity, error) {
	return r.detail, r.err
}

func (r *stubUserIdentityScopeRepo) GetPageListByCond(ctx context.Context, cond *dao.UserIdentityCond) (model.UserIdentityEntityList, int64, error) {
	clone := *cond
	if cond.BaseCond != nil {
		base := *cond.BaseCond
		clone.BaseCond = &base
	}
	r.lastCond = &clone
	return r.pageList, r.total, r.err
}

func (r *stubUserIdentityScopeRepo) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	return r.err
}

func (r *stubUserIdentityScopeRepo) Delete(ctx context.Context, id uint, deletedBy uint) error {
	return r.err
}

func (r *stubUserIdentityScopeRepo) UpdateMap(ctx context.Context, id uint, updates map[string]any) error {
	return r.err
}

func installUserIdentityScopeRepo(t *testing.T, repo userIdentityRepository) {
	t.Helper()
	prev := newUserIdentityRepo
	prevSvc := newPersonIdentitySvc
	newUserIdentityRepo = func() userIdentityRepository { return repo }
	newPersonIdentitySvc = func() delegatedPersonIdentitySvc {
		return &stubDelegatedPersonIdentitySvc{repo: repo, userRepo: newUserIdentityUserRepo()}
	}
	t.Cleanup(func() {
		newUserIdentityRepo = prev
		newPersonIdentitySvc = prevSvc
	})
}

func installUserIdentityUserResolver(t *testing.T, repo userIdentityUserResolver) {
	t.Helper()
	prev := newUserIdentityUserRepo
	newUserIdentityUserRepo = func() userIdentityUserResolver { return repo }
	t.Cleanup(func() { newUserIdentityUserRepo = prev })
}

var _ userObjectScopeRepository = (*stubUserObjectScopeRepo)(nil)
var _ userQueryRepository = (*stubUserObjectScopeRepo)(nil)
var _ userLoginLogDetailRepository = (*stubUserLoginLogDetailRepo)(nil)
var _ userIdentityRepository = (*stubUserIdentityScopeRepo)(nil)
