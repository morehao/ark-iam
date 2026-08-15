package svctenant

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

func newTenantScopeGinCtx(tenantID uint) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	return ctx
}

func TestDepartmentDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{})
	ctx := newTenantScopeGinCtx(11)

	dept := &model.DepartmentEntity{TenantID: 22, Name: "other-tenant-dept"}
	if err := db.Create(dept).Error; err != nil {
		t.Fatalf("seed department: %v", err)
	}

	svc := &departmentSvc{}
	resp, err := svc.Detail(ctx, &dtotenant.DepartmentDetailReq{DepartmentID: dept.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant detail to fail, resp=%+v", resp)
	}
}

func TestDepartmentPageListUsesContextTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{})
	ctx := newTenantScopeGinCtx(33)

	// 同名校部门分别属于上下文租户 33 与请求体租户 99，
	// 若服务误用请求体的 TenantID=99，则 99 租户的部门也会被查出。
	if err := db.Create(&model.DepartmentEntity{TenantID: 33, Name: "shared"}).Error; err != nil {
		t.Fatalf("seed tenant 33 department: %v", err)
	}
	if err := db.Create(&model.DepartmentEntity{TenantID: 99, Name: "shared"}).Error; err != nil {
		t.Fatalf("seed tenant 99 department: %v", err)
	}

	svc := &departmentSvc{}
	resp, err := svc.PageList(ctx, &dtotenant.DepartmentPageListReq{TenantID: 99, Name: "shared"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected only tenant 33 department, got total=%d list=%+v", resp.Total, resp.List)
	}
	if resp.List[0].TenantID != 33 {
		t.Fatalf("expected tenant 33 from context, got %+v", resp.List[0])
	}
}

func TestDepartmentTreeUsesContextTenant(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.DepartmentEntity{})
	ctx := newTenantScopeGinCtx(34)

	// 根部门分别属于上下文租户 34 与请求体租户 99，
	// 若服务误用请求体的 TenantID=99，则 99 租户的根部门也会出现在树中。
	if err := db.Create(&model.DepartmentEntity{TenantID: 34, ParentID: 0, Name: "root34"}).Error; err != nil {
		t.Fatalf("seed tenant 34 department: %v", err)
	}
	if err := db.Create(&model.DepartmentEntity{TenantID: 99, ParentID: 0, Name: "root99"}).Error; err != nil {
		t.Fatalf("seed tenant 99 department: %v", err)
	}

	svc := &departmentSvc{}
	resp, err := svc.Tree(ctx, &dtotenant.DepartmentTreeReq{TenantID: 99})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected only tenant 34 department in tree, got %+v", resp.List)
	}
	if resp.List[0].TenantID != 34 {
		t.Fatalf("expected tenant 34 from context, got %+v", resp.List[0])
	}
}

func TestSystemDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.SystemEntity{})
	ctx := newTenantScopeGinCtx(61)

	system := &model.SystemEntity{TenantID: 90, Key: "demo", Value: json.RawMessage(`{"enabled":true}`)}
	if err := db.Create(system).Error; err != nil {
		t.Fatalf("seed system: %v", err)
	}

	svc := &systemSvc{}
	resp, err := svc.Detail(ctx, &dtotenant.SystemDetailReq{SystemID: system.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant system detail to fail, resp=%+v", resp)
	}
}

func TestLogDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.LogEntity{})
	ctx := newTenantScopeGinCtx(71)

	logEntity := &model.LogEntity{TenantID: 91, Key: "audit", Payload: json.RawMessage(`{"trace":"x"}`)}
	if err := db.Create(logEntity).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	svc := &logSvc{}
	resp, err := svc.Detail(ctx, &dtotenant.LogDetailReq{LogID: logEntity.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant log detail to fail, resp=%+v", resp)
	}
}
