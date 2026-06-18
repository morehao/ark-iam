package config

import "testing"

func TestSSOCookieDomainDefaultsToLocalhost(t *testing.T) {
	o := OIDC{CookieDomain: ""}
	if got := o.SSOCookieDomain(); got != "" {
		t.Fatalf("expected empty default cookie domain, got %q", got)
	}
}

func TestSSOCookieDomainUsesConfiguredValue(t *testing.T) {
	o := OIDC{CookieDomain: "example.com"}
	if got := o.SSOCookieDomain(); got != "example.com" {
		t.Fatalf("expected 'example.com', got %q", got)
	}
}
