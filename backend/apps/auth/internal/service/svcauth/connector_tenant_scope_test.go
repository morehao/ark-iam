package svcauth

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

func TestConnectorDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(71))

	repo := &stubConnectorScopeRepo{detail: &model.ConnectorEntity{Model: gorm.Model{ID: 10}, TenantID: 99}}
	installConnectorScopeRepo(t, repo)

	svc := &connectorSvc{}
	_, err := svc.Detail(ginCtx, &dtoauth.ConnectorDetailReq{ConnectorID: 10})
	if err == nil {
		t.Fatalf("expected cross-tenant connector detail to fail")
	}
}

func TestConnectorPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(72))

	repo := &stubConnectorScopeRepo{}
	installConnectorScopeRepo(t, repo)

	svc := &connectorSvc{}
	_, _ = svc.PageList(ginCtx, &dtoauth.ConnectorPageListReq{TenantID: 99})
	if repo.lastCond == nil || repo.lastCond.TenantID != 72 {
		t.Fatalf("expected tenant 72 from context, got %+v", repo.lastCond)
	}
}

type stubConnectorScopeRepo struct {
	detail   *model.ConnectorEntity
	pageList model.ConnectorEntityList
	total    int64
	err      error
	lastCond *dao.ConnectorCond
}

func (r *stubConnectorScopeRepo) GetByID(ctx context.Context, id uint) (*model.ConnectorEntity, error) {
	return r.detail, r.err
}

func (r *stubConnectorScopeRepo) GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ConnectorEntityList, int64, error) {
	typed, _ := cond.(*dao.ConnectorCond)
	clone := *typed
	if typed.BaseCond != nil {
		base := *typed.BaseCond
		clone.BaseCond = &base
	}
	r.lastCond = &clone
	return r.pageList, r.total, r.err
}

func installConnectorScopeRepo(t *testing.T, repo connectorScopeRepository) {
	t.Helper()
	prev := newConnectorScopeRepo
	newConnectorScopeRepo = func() connectorScopeRepository { return repo }
	t.Cleanup(func() { newConnectorScopeRepo = prev })
}

var _ connectorScopeRepository = (*stubConnectorScopeRepo)(nil)
