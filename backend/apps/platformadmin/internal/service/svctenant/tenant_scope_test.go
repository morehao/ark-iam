package svctenant

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func TestDepartmentDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(11))

	repo := &stubDepartmentScopeRepo{
		detail: &model.DepartmentEntity{Model: gorm.Model{ID: 7}, TenantID: 22},
	}
	installDepartmentScopeRepo(t, repo)

	svc := &departmentSvc{}
	resp, err := svc.Detail(ginCtx, &dtotenant.DepartmentDetailReq{DepartmentID: 7})
	if err == nil {
		t.Fatalf("expected cross-tenant detail to fail, resp=%+v", resp)
	}
}

func TestDepartmentPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(33))

	repo := &stubDepartmentScopeRepo{recordTarget: "page"}
	installDepartmentScopeRepo(t, repo)

	svc := &departmentSvc{}
	_, _ = svc.PageList(ginCtx, &dtotenant.DepartmentPageListReq{TenantID: 99})
	if repo.lastPageCond == nil || repo.lastPageCond.TenantID != 33 {
		t.Fatalf("expected tenant 33 from context, got %+v", repo.lastPageCond)
	}
}

func TestDepartmentTreeUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(34))

	repo := &stubDepartmentScopeRepo{recordTarget: "tree"}
	installDepartmentScopeRepo(t, repo)

	svc := &departmentSvc{}
	_, _ = svc.Tree(ginCtx, &dtotenant.DepartmentTreeReq{TenantID: 99})
	if repo.lastTreeCond == nil || repo.lastTreeCond.TenantID != 34 {
		t.Fatalf("expected tenant 34 from context, got %+v", repo.lastTreeCond)
	}
}

func TestSystemDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(61))

	value, _ := json.Marshal(map[string]any{"enabled": true})
	repo := &stubSystemScopeRepo{
		detail: &model.SystemEntity{Model: gorm.Model{ID: 10}, TenantID: 90, Key: "demo", Value: value},
	}
	installSystemScopeRepo(t, repo)

	svc := &systemSvc{}
	resp, err := svc.Detail(ginCtx, &dtotenant.SystemDetailReq{SystemID: 10})
	if err == nil {
		t.Fatalf("expected cross-tenant system detail to fail, resp=%+v", resp)
	}
}

func TestLogDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(71))

	payload, _ := json.Marshal(map[string]any{"trace": "x"})
	repo := &stubLogScopeRepo{
		detail: &model.LogEntity{Model: gorm.Model{ID: 11, CreatedAt: time.Unix(1700000000, 0)}, TenantID: 91, Key: "audit", Payload: payload},
	}
	installLogScopeRepo(t, repo)

	svc := &logSvc{}
	resp, err := svc.Detail(ginCtx, &dtotenant.LogDetailReq{LogID: 11})
	if err == nil {
		t.Fatalf("expected cross-tenant log detail to fail, resp=%+v", resp)
	}
}

type stubDepartmentScopeRepo struct {
	detail       *model.DepartmentEntity
	pageList     model.DepartmentEntityList
	total        int64
	err          error
	lastPageCond *dao.DepartmentCond
	lastTreeCond *dao.DepartmentCond
	recordTarget string
}

func (r *stubDepartmentScopeRepo) GetByID(ctx context.Context, id uint) (*model.DepartmentEntity, error) {
	return r.detail, r.err
}

func (r *stubDepartmentScopeRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.DepartmentEntityList, int64, error) {
	typed, _ := cond.(*dao.DepartmentCond)
	clone := cloneDepartmentCond(typed)
	if r.recordTarget == "tree" {
		r.lastTreeCond = clone
	} else {
		r.lastPageCond = clone
	}
	return r.pageList, r.total, r.err
}

func installDepartmentScopeRepo(t *testing.T, repo departmentScopeRepository) {
	t.Helper()
	prev := newDepartmentScopeRepo
	newDepartmentScopeRepo = func() departmentScopeRepository {
		return repo
	}
	t.Cleanup(func() {
		newDepartmentScopeRepo = prev
	})
}

func cloneDepartmentCond(cond *dao.DepartmentCond) *dao.DepartmentCond {
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

type stubSystemScopeRepo struct {
	detail *model.SystemEntity
	err    error
}

func (r *stubSystemScopeRepo) GetByID(ctx context.Context, id uint) (*model.SystemEntity, error) {
	return r.detail, r.err
}

func installSystemScopeRepo(t *testing.T, repo systemScopeRepository) {
	t.Helper()
	prev := newSystemScopeRepo
	newSystemScopeRepo = func() systemScopeRepository {
		return repo
	}
	t.Cleanup(func() {
		newSystemScopeRepo = prev
	})
}

type stubLogScopeRepo struct {
	detail *model.LogEntity
	err    error
}

func (r *stubLogScopeRepo) GetByID(ctx context.Context, id uint) (*model.LogEntity, error) {
	return r.detail, r.err
}

func installLogScopeRepo(t *testing.T, repo logScopeRepository) {
	t.Helper()
	prev := newLogScopeRepo
	newLogScopeRepo = func() logScopeRepository {
		return repo
	}
	t.Cleanup(func() {
		newLogScopeRepo = prev
	})
}

var _ departmentScopeRepository = (*stubDepartmentScopeRepo)(nil)
var _ systemScopeRepository = (*stubSystemScopeRepo)(nil)
var _ logScopeRepository = (*stubLogScopeRepo)(nil)
