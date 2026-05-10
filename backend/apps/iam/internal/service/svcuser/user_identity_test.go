package svcuser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

func TestUserIdentityPageListPassesFiltersToDAOAndKeepsDAOTotal(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(23))
	repo := &stubUserIdentityRepo{
		pageList: model.UserIdentityEntityList{
			{
				Model:           gorm.Model{ID: 1, UpdatedAt: time.Unix(1700000000, 0)},
				TenantID:        23,
				UserID:          101,
				Issuer:          "issuer-a",
				ExternalSubject: "external-1",
				Detail:          []byte(`{"name":"first"}`),
			},
		},
		total: 7,
	}
	installUserIdentityRepo(t, repo)

	svc := &userIdentitySvc{}
	resp, err := svc.PageList(ginCtx, &dtouser.UserIdentityPageListReq{
		PageQuery:  gobject.PageQuery{Page: 2, PageSize: 5},
		TenantID:   23,
		UserID:     101,
		Issuer:     "issuer-a",
		IdentityID: "external-1",
	})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected DAO condition to be captured")
	}
	if repo.lastCond.TenantID != 23 {
		t.Fatalf("expected tenant id 23 from context, got %d", repo.lastCond.TenantID)
	}
	if repo.lastCond.UserID != 101 {
		t.Fatalf("expected user id 101, got %d", repo.lastCond.UserID)
	}
	if repo.lastCond.Issuer != "issuer-a" {
		t.Fatalf("expected issuer issuer-a, got %q", repo.lastCond.Issuer)
	}
	if repo.lastCond.ExternalSubject != "external-1" {
		t.Fatalf("expected external subject external-1, got %q", repo.lastCond.ExternalSubject)
	}
	if repo.lastCond.BaseCond == nil || repo.lastCond.BaseCond.Page != 2 || repo.lastCond.BaseCond.PageSize != 5 {
		t.Fatalf("expected page condition {2,5}, got %#v", repo.lastCond.BaseCond)
	}
	if resp.Total != 7 {
		t.Fatalf("expected total 7 from DAO, got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(resp.List))
	}
}

func TestUserIdentityGetByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(45))
	repo := &stubUserIdentityRepo{
		pageList: model.UserIdentityEntityList{
			{
				Model:           gorm.Model{ID: 2, UpdatedAt: time.Unix(1700000001, 0)},
				TenantID:        45,
				UserID:          202,
				Issuer:          "issuer-b",
				ExternalSubject: "external-2",
				Detail:          []byte(`{"name":"second"}`),
			},
		},
		total: 3,
	}
	installUserIdentityRepo(t, repo)

	svc := &userIdentitySvc{}
	resp, err := svc.GetByUser(ginCtx, &dtouser.UserIdentityByUserReq{UserID: 202})
	if err != nil {
		t.Fatalf("GetByUser returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected DAO condition to be captured")
	}
	if repo.lastCond.TenantID != 45 {
		t.Fatalf("expected tenant id 45, got %d", repo.lastCond.TenantID)
	}
	if repo.lastCond.UserID != 202 {
		t.Fatalf("expected user id 202, got %d", repo.lastCond.UserID)
	}
	if repo.lastCond.BaseCond != nil {
		t.Fatalf("expected no pagination base condition, got %#v", repo.lastCond.BaseCond)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total 3 from DAO, got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(resp.List))
	}
}

type stubUserIdentityRepo struct {
	pageList model.UserIdentityEntityList
	total    int64
	err      error
	lastCond *dao.UserIdentityCond
}

func (r *stubUserIdentityRepo) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	return errors.New("unexpected call to Insert")
}

func (r *stubUserIdentityRepo) GetByID(ctx context.Context, id uint) (*model.UserIdentityEntity, error) {
	return nil, errors.New("unexpected call to GetByID")
}

func (r *stubUserIdentityRepo) Delete(ctx context.Context, id uint, deletedBy uint) error {
	return errors.New("unexpected call to Delete")
}

func (r *stubUserIdentityRepo) UpdateMap(ctx context.Context, id uint, updates map[string]any) error {
	return errors.New("unexpected call to UpdateMap")
}

func (r *stubUserIdentityRepo) GetPageListByCond(ctx context.Context, cond *dao.UserIdentityCond) (model.UserIdentityEntityList, int64, error) {
	r.lastCond = cloneUserIdentityCond(cond)
	return r.pageList, r.total, r.err
}

func installUserIdentityRepo(t *testing.T, repo userIdentityRepository) {
	t.Helper()
	prev := newUserIdentityRepo
	newUserIdentityRepo = func() userIdentityRepository {
		return repo
	}
	t.Cleanup(func() {
		newUserIdentityRepo = prev
	})
}

func cloneUserIdentityCond(cond *dao.UserIdentityCond) *dao.UserIdentityCond {
	if cond == nil {
		return nil
	}
	clone := *cond
	if cond.BaseCond != nil {
		base := *cond.BaseCond
		clone.BaseCond = &base
	}
	return &clone
}

var _ userIdentityRepository = (*stubUserIdentityRepo)(nil)

var _ = genericdao.BaseCond{}
