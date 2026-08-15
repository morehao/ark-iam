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

func newTenantScopeGinCtx(tenantID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	return ctx
}

func TestSystemDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.SystemEntity{})
	ctx := newTenantScopeGinCtx("61")

	system := &model.SystemEntity{TenantID: "90", Key: "demo", Value: json.RawMessage(`{"enabled":true}`)}
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
	ctx := newTenantScopeGinCtx("71")

	logEntity := &model.LogEntity{TenantID: "91", Key: "audit", Payload: json.RawMessage(`{"trace":"x"}`)}
	if err := db.Create(logEntity).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	svc := &logSvc{}
	resp, err := svc.Detail(ctx, &dtotenant.LogDetailReq{LogID: logEntity.ID})
	if err == nil {
		t.Fatalf("expected cross-tenant log detail to fail, resp=%+v", resp)
	}
}
