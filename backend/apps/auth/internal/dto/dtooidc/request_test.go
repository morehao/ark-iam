package dtooidc

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOIDCLoginReqBindingRequiresAllFields(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	body := []byte(`{"identifier":"person@example.com","password":"Password1"}`)
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	ginCtx.Request = req

	var loginReq OIDCLoginReq
	if err := ginCtx.ShouldBindJSON(&loginReq); err == nil {
		t.Fatal("expected missing authRequestID to fail binding")
	}
}
