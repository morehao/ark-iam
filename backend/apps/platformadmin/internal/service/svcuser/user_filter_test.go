package svcuser

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gobject"
	"gorm.io/gorm"
)

func TestUserPageListPassesIsSuspendedZeroFilterToDAOCondition(t *testing.T) {
	isSuspended := int8(0)
	repo := &stubUserQueryRepo{
		pageList: model.UserEntityList{
			{Model: gorm.Model{ID: 1, UpdatedAt: time.Unix(1700000000, 0)}, TenantID: 23, Name: "active-user", Profile: []byte(`{}`), CustomData: []byte(`{}`), IsSuspended: 0},
		},
		total: 1,
	}
	installUserQueryRepo(t, repo)

	ginCtx, _ := gin.CreateTestContext(nil)
	svc := &userSvc{}
	_, err := svc.PageList(ginCtx, &dtouser.UserPageListReq{
		PageQuery:   gobject.PageQuery{Page: 1, PageSize: 10},
		TenantID:    23,
		IsSuspended: &isSuspended,
	})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if repo.lastCond == nil || repo.lastCond.IsSuspended == nil {
		t.Fatalf("expected DAO condition to capture isSuspended filter")
	}
	if *repo.lastCond.IsSuspended != 0 {
		t.Fatalf("expected isSuspended filter 0, got %d", *repo.lastCond.IsSuspended)
	}
}

type stubUserQueryRepo struct {
	pageList model.UserEntityList
	total    int64
	err      error
	lastCond *dao.UserCond
}

func (r *stubUserQueryRepo) GetPageListByCond(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, int64, error) {
	r.lastCond = cloneUserCond(cond)
	return r.pageList, r.total, r.err
}

func installUserQueryRepo(t *testing.T, repo userQueryRepository) {
	t.Helper()
	prev := newUserQueryRepo
	newUserQueryRepo = func() userQueryRepository {
		return repo
	}
	t.Cleanup(func() {
		newUserQueryRepo = prev
	})
}

func cloneUserCond(cond *dao.UserCond) *dao.UserCond {
	if cond == nil {
		return nil
	}
	clone := *cond
	if cond.BaseCond != nil {
		base := *cond.BaseCond
		clone.BaseCond = &base
	}
	if cond.IsSuspended != nil {
		value := *cond.IsSuspended
		clone.IsSuspended = &value
	}
	return &clone
}

var _ userQueryRepository = (*stubUserQueryRepo)(nil)
