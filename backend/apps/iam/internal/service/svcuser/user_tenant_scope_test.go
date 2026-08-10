package svcuser

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

func TestGetUserLoginLogByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(45))
	repo := &stubUserLoginLogQueryRepo{
		pageList: model.UserLoginLogEntityList{
			{
				Model:     gorm.Model{ID: 2, UpdatedAt: time.Unix(1700000001, 0)},
				TenantID:  45,
				UserID:    202,
				LoginIP:   "127.0.0.1",
				UserAgent: "chrome",
				LoginTime: time.Unix(1700000000, 0),
			},
		},
		total: 3,
	}
	installUserLoginLogQueryRepo(t, repo)

	svc := &userSvc{}
	resp, err := svc.GetUserLoginLogByUser(ginCtx, &dtouser.UserLoginLogByUserReq{UserID: 202})
	if err != nil {
		t.Fatalf("GetUserLoginLogByUser returned error: %v", err)
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

func TestGetUserDepartmentByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(46))
	repo := &stubUserDepartmentQueryRepo{
		pageList: model.UserDepartmentEntityList{
			{
				Model:        gorm.Model{ID: 3, UpdatedAt: time.Unix(1700000002, 0)},
				TenantID:     46,
				UserID:       303,
				DepartmentID: 404,
				IsPrimary:    1,
			},
		},
		total: 4,
	}
	installUserDepartmentQueryRepo(t, repo)

	svc := &userSvc{}
	resp, err := svc.GetUserDepartmentByUser(ginCtx, &dtouser.UserDepartmentByUserReq{UserID: 303})
	if err != nil {
		t.Fatalf("GetUserDepartmentByUser returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected DAO condition to be captured")
	}
	if repo.lastCond.TenantID != 46 {
		t.Fatalf("expected tenant id 46, got %d", repo.lastCond.TenantID)
	}
	if repo.lastCond.UserID != 303 {
		t.Fatalf("expected user id 303, got %d", repo.lastCond.UserID)
	}
	if repo.lastCond.BaseCond != nil {
		t.Fatalf("expected no pagination base condition, got %#v", repo.lastCond.BaseCond)
	}
	if resp.Total != 4 {
		t.Fatalf("expected total 4 from DAO, got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(resp.List))
	}
}

type stubUserLoginLogQueryRepo struct {
	pageList model.UserLoginLogEntityList
	total    int64
	err      error
	lastCond *dao.UserLoginLogCond
}

func (r *stubUserLoginLogQueryRepo) GetPageListByCond(ctx context.Context, cond *dao.UserLoginLogCond) (model.UserLoginLogEntityList, int64, error) {
	r.lastCond = cloneUserLoginLogCond(cond)
	return r.pageList, r.total, r.err
}

func installUserLoginLogQueryRepo(t *testing.T, repo userLoginLogQueryRepository) {
	t.Helper()
	prev := newUserLoginLogQueryRepo
	newUserLoginLogQueryRepo = func() userLoginLogQueryRepository {
		return repo
	}
	t.Cleanup(func() {
		newUserLoginLogQueryRepo = prev
	})
}

func cloneUserLoginLogCond(cond *dao.UserLoginLogCond) *dao.UserLoginLogCond {
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

type stubUserDepartmentQueryRepo struct {
	pageList model.UserDepartmentEntityList
	total    int64
	err      error
	lastCond *dao.UserDepartmentCond
}

func (r *stubUserDepartmentQueryRepo) GetPageListByCond(ctx context.Context, cond *dao.UserDepartmentCond) (model.UserDepartmentEntityList, int64, error) {
	r.lastCond = cloneUserDepartmentCond(cond)
	return r.pageList, r.total, r.err
}

func installUserDepartmentQueryRepo(t *testing.T, repo userDepartmentQueryRepository) {
	t.Helper()
	prev := newUserDepartmentQueryRepo
	newUserDepartmentQueryRepo = func() userDepartmentQueryRepository {
		return repo
	}
	t.Cleanup(func() {
		newUserDepartmentQueryRepo = prev
	})
}

func cloneUserDepartmentCond(cond *dao.UserDepartmentCond) *dao.UserDepartmentCond {
	if cond == nil {
		return nil
	}
	clone := *cond
	if cond.BaseCond != nil {
		base := *cond.BaseCond
		clone.BaseCond = &base
	}
	if cond.IsPrimary != nil {
		value := *cond.IsPrimary
		clone.IsPrimary = &value
	}
	return &clone
}

var _ userLoginLogQueryRepository = (*stubUserLoginLogQueryRepo)(nil)

var _ userDepartmentQueryRepository = (*stubUserDepartmentQueryRepo)(nil)
