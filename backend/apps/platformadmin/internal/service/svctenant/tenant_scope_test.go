package svctenant

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/testutil"
)

// newTenantScopeGinCtx 构造带租户作用域的测试 gin.Context；tenantID 为空表示平台侧
// 「全部租户」视角（按主键列表、建租户链路）。同时挂上真实 Request（生产链路必有），
// 避免服务内 ctx.Request.Context() 空指针。
func newTenantScopeGinCtx(tenantID string) *gin.Context {
	return testutil.NewGinCtx(tenantID, "")
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
