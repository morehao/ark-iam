package svcauth

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func TestConnectorDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, "71")

	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})
	if err := db.Create(&model.ConnectorEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: "10"}},
		TenantID:     "99",
		Name:         "cross",
		Config:       []byte("{}"),
		ClaimMapping: []byte("{}"),
		DomainPolicy: []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := &connectorSvc{}
	_, err := svc.Detail(ginCtx, &dtoauth.ConnectorDetailReq{ConnectorID: "10"})
	if err == nil {
		t.Fatalf("expected cross-tenant connector detail to fail")
	}
}

func TestConnectorPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, "72")

	db := testutil.SetupSQLite(t, &model.ConnectorEntity{})
	if err := db.Create(&model.ConnectorEntity{
		TenantID:     "72",
		Name:         "tenant-72",
		Config:       []byte("{}"),
		ClaimMapping: []byte("{}"),
		DomainPolicy: []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("seed tenant 72: %v", err)
	}
	if err := db.Create(&model.ConnectorEntity{
		TenantID:     "99",
		Name:         "tenant-99",
		Config:       []byte("{}"),
		ClaimMapping: []byte("{}"),
		DomainPolicy: []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("seed tenant 99: %v", err)
	}

	svc := &connectorSvc{}
	// 请求里的 TenantID 为 99，但服务必须以上下文租户 72 为准
	resp, err := svc.PageList(ginCtx, &dtoauth.ConnectorPageListReq{TenantID: "99"})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1 (context tenant), got %d", resp.Total)
	}
	if len(resp.List) != 1 || resp.List[0].Name != "tenant-72" {
		t.Fatalf("expected tenant 72 connector from context, got %+v", resp.List)
	}
}
