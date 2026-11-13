package router_test

import "testing"

func assertRouteRegistered(t *testing.T, routes map[string]map[string]bool, method, path string) {
	t.Helper()
	if !routes[path][method] {
		t.Fatalf("expected route %s %s to be registered", method, path)
	}
}

func assertRouteMissing(t *testing.T, routes map[string]map[string]bool, method, path string) {
	t.Helper()
	if routes[path][method] {
		t.Fatalf("expected route %s %s to be removed", method, path)
	}
}
