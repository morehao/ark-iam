package svcoidc

import (
	"net/http"
	"testing"

	"github.com/morehao/ark-iam/pkg/oidcctx"
)

// TestCustomIssuerFromRequest 验证 op IssuerFromRequest 按请求 ctx 的 pkg/oidcctx 返回 Host issuer，
// 未命中回退静态 issuer（P0 多 issuer 在 OP 协议栈生效的关键接线）。
func TestCustomIssuerFromRequest(t *testing.T) {
	build := customIssuerFromRequest("http://platform.test/oidc")
	ifn, err := build(false)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	// 无 oidcctx → 静态 issuer
	if got := ifn(&http.Request{}); got != "http://platform.test/oidc" {
		t.Fatalf("fallback = %q", got)
	}
	// 带 oidcctx(D issuer) → 返回 D issuer
	req, _ := http.NewRequest(http.MethodGet, "http://login.acme.test/oidc/.well-known/openid-configuration", nil)
	req = req.WithContext(oidcctx.WithIssuer(req.Context(), "https://login.acme.test/oidc"))
	if got := ifn(req); got != "https://login.acme.test/oidc" {
		t.Fatalf("per-request issuer = %q, want D issuer", got)
	}
}
