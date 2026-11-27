package oidcctx

import (
	"context"
	"testing"
)

func TestIssuerRoundTrip(t *testing.T) {
	ctx := context.Background()
	if _, ok := IssuerFromContext(ctx); ok {
		t.Fatal("expected no issuer on empty ctx")
	}
	ctx = WithIssuer(ctx, "https://login.acme.com/oidc")
	got, ok := IssuerFromContext(ctx)
	if !ok || got != "https://login.acme.com/oidc" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	// 空 issuer 写入应保持无值（fail-closed 语义：不回退中央）
	empty := WithIssuer(context.Background(), "")
	if _, ok := IssuerFromContext(empty); ok {
		t.Fatal("empty issuer should not be stored")
	}
}
