package svctenant

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

// newTenantScopeGinCtx 构造带租户上下文的测试 gin.Context；
// 同时挂上真实 Request（生产链路必有），避免服务内 ctx.Request.Context() 空指针。
func newTenantScopeGinCtx(tenantID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	return ctx
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
