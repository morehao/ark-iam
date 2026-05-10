package svcapplication

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

func TestApplicationDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(61))

	repo := &stubApplicationScopeRepo{detail: &model.ApplicationEntity{Model: gorm.Model{ID: 9}, TenantID: 99}}
	installApplicationScopeRepo(t, repo)

	svc := &applicationSvc{}
	_, err := svc.Detail(ginCtx, &dtoapplication.ApplicationDetailReq{ApplicationID: 9})
	if err == nil {
		t.Fatalf("expected cross-tenant application detail to fail")
	}
}

func TestApplicationPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(62))

	repo := &stubApplicationScopeRepo{}
	installApplicationScopeRepo(t, repo)

	svc := &applicationSvc{}
	_, _ = svc.PageList(ginCtx, &dtoapplication.ApplicationPageListReq{TenantID: 99})
	if repo.lastCond == nil || repo.lastCond.TenantID != 62 {
		t.Fatalf("expected tenant 62 from context, got %+v", repo.lastCond)
	}
}

func TestDeleteSecretRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(63))
	ginCtx.Set(gcontext.KeyUserID, uint(1003))

	repo := &stubApplicationScopeRepo{secret: &model.ApplicationSecretEntity{Model: gorm.Model{ID: 7}, TenantID: 98}}
	installApplicationScopeRepo(t, repo)

	svc := &applicationSvc{}
	err := svc.DeleteSecret(ginCtx, &dtoapplication.DeleteApplicationSecretReq{SecretID: 7})
	if err == nil {
		t.Fatalf("expected cross-tenant secret delete to fail")
	}
	if repo.deletedSecretID != 0 {
		t.Fatalf("expected no secret delete for cross-tenant entity")
	}
}

type stubApplicationScopeRepo struct {
	detail          *model.ApplicationEntity
	pageList        model.ApplicationEntityList
	total           int64
	secret          *model.ApplicationSecretEntity
	err             error
	lastCond        *dao.ApplicationCond
	deletedSecretID uint
}

func (r *stubApplicationScopeRepo) GetByID(ctx context.Context, id uint) (*model.ApplicationEntity, error) {
	return r.detail, r.err
}

func (r *stubApplicationScopeRepo) GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationEntityList, int64, error) {
	typed, _ := cond.(*dao.ApplicationCond)
	clone := *typed
	if typed.BaseCond != nil {
		base := *typed.BaseCond
		clone.BaseCond = &base
	}
	r.lastCond = &clone
	return r.pageList, r.total, r.err
}

func (r *stubApplicationScopeRepo) GetSecretByID(ctx context.Context, id uint) (*model.ApplicationSecretEntity, error) {
	return r.secret, r.err
}

func (r *stubApplicationScopeRepo) DeleteSecret(ctx context.Context, id uint, userID uint) error {
	r.deletedSecretID = id
	return r.err
}

func installApplicationScopeRepo(t *testing.T, repo applicationScopeRepository) {
	t.Helper()
	prev := newApplicationScopeRepo
	newApplicationScopeRepo = func() applicationScopeRepository { return repo }
	t.Cleanup(func() { newApplicationScopeRepo = prev })
}

var _ applicationScopeRepository = (*stubApplicationScopeRepo)(nil)
