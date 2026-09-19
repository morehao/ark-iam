package identity

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/sdk/rp"
)

// newTestCtx 构造可写的 gin 测试上下文（gin 的 CreateTestContext 不自动带 Request）。
func newTestCtx() *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return ctx
}

func TestClientIDFromContext(t *testing.T) {
	ctx := newTestCtx()

	// 未写入：返回空字符串（调用方 fail-closed 的依据）。
	if got := ClientIDFromContext(ctx); got != "" {
		t.Fatalf("unwritten client_id = %q, want empty", got)
	}

	SetClientID(ctx, "platform_admin_web")
	if got := ClientIDFromContext(ctx); got != "platform_admin_web" {
		t.Fatalf("client_id = %q, want platform_admin_web", got)
	}

	// 覆盖写：后写生效（同一请求内重复写入不应累积）。
	SetClientID(ctx, "tenant_admin_web")
	if got := ClientIDFromContext(ctx); got != "tenant_admin_web" {
		t.Fatalf("client_id after overwrite = %q, want tenant_admin_web", got)
	}
}

func TestClientIDFromContext_NilCtx(t *testing.T) {
	SetClientID(nil, "ignored")
	if got := ClientIDFromContext(nil); got != "" {
		t.Fatalf("nil ctx client_id = %q, want empty", got)
	}
}

func TestOIDCIdentityFromContext(t *testing.T) {
	ctx := newTestCtx()

	// 未经过 OIDC 鉴权：返回 nil，调用方不得据此放行。
	if got := OIDCIdentityFromContext(ctx); got != nil {
		t.Fatalf("unwritten identity = %v, want nil", got)
	}

	ident := &rp.Identity{PersonID: "person:1", TenantID: "t-1", ClientID: "app_web", UserID: "u-1"}
	SetOIDCIdentity(ctx, ident)

	got := OIDCIdentityFromContext(ctx)
	if got == nil {
		t.Fatal("identity = nil, want the injected pointer")
	}
	if got != ident {
		t.Fatalf("identity = %p, want same pointer %p", got, ident)
	}
	if got.TenantID != "t-1" || got.UserID != "u-1" {
		t.Fatalf("identity fields = %+v, want tenant t-1 / user u-1", got)
	}
}

func TestOIDCIdentityFromContext_WrongTypeIsNil(t *testing.T) {
	ctx := newTestCtx()
	// 键被写入非 *rp.Identity 值（例如测试或中间件误用）时必须返回 nil，绝不 panic。
	ctx.Set(contextKeyOIDCIdentity, "not-an-identity")

	if got := OIDCIdentityFromContext(ctx); got != nil {
		t.Fatalf("identity with wrong type = %v, want nil", got)
	}
}

func TestOIDCIdentityFromContext_NilCtx(t *testing.T) {
	SetOIDCIdentity(nil, &rp.Identity{PersonID: "person:1"})
	if got := OIDCIdentityFromContext(nil); got != nil {
		t.Fatalf("nil ctx identity = %v, want nil", got)
	}
}
