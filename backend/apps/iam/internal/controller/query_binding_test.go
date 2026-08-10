package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestQueryBindingUserDetailReqUsesFormTags(t *testing.T) {
	engine := gin.New()
	var got dtouser.UserDetailReq
	engine.GET("/user/detail", func(ctx *gin.Context) {
		if err := ctx.ShouldBindQuery(&got); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/user/detail?userID=12", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
	}
	if got.UserID != 12 {
		t.Fatalf("expected userID 12, got %d", got.UserID)
	}
}

func TestQueryBindingTenantDetailReqUsesFormTags(t *testing.T) {
	engine := gin.New()
	var got dtotenant.TenantDetailReq
	engine.GET("/tenant/detail", func(ctx *gin.Context) {
		if err := ctx.ShouldBindQuery(&got); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/tenant/detail?tenantID=34", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
	}
	if got.TenantID != 34 {
		t.Fatalf("expected tenantID 34, got %d", got.TenantID)
	}
}

func TestQueryBindingConnectorDetailReqUsesFormTags(t *testing.T) {
	engine := gin.New()
	var got dtoauth.ConnectorDetailReq
	engine.GET("/connector/detail", func(ctx *gin.Context) {
		if err := ctx.ShouldBindQuery(&got); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/connector/detail?connectorId=56", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
	}
	if got.ConnectorID != 56 {
		t.Fatalf("expected connectorId 56, got %d", got.ConnectorID)
	}
}

func TestQueryBindingSessionListReqUsesFormTags(t *testing.T) {
	engine := gin.New()
	var got dtouser.SessionListReq
	engine.GET("/user/sessions", func(ctx *gin.Context) {
		if err := ctx.ShouldBindQuery(&got); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/user/sessions?page=3&pageSize=20", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, resp.Code, resp.Body.String())
	}
	if got.Page != 3 || got.PageSize != 20 {
		t.Fatalf("unexpected bound req: %#v", got)
	}
}
