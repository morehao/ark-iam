package ctroidc

import "testing"

// TestOIDCStripPrefix 验证按请求路径动态剥离 OIDC 挂载前缀，
// 覆盖空前缀（根挂载）与任意非空前缀（内嵌 /iam 形态）。
func TestOIDCStripPrefix(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{"根挂载 keys", "/oidc/keys", "/oidc"},
		{"根挂载 discovery", "/oidc/.well-known/openid-configuration", "/oidc"},
		{"内嵌 /iam authorize", "/iam/oidc/authorize", "/iam/oidc"},
		{"内嵌 /iam token", "/iam/oidc/oauth/token", "/iam/oidc"},
		{"内嵌 /iam keys", "/iam/oidc/keys", "/iam/oidc"},
		{"多层前缀", "/x/y/oidc/userinfo", "/x/y/oidc"},
		{"无 oidc 段", "/healthz", ""},
		{"相似前缀不误伤", "/oidcxx/keys", ""},
		{"前缀含 oidc 取最后一个", "/oidc/app/oidc/keys", "/oidc/app/oidc"},
	}
	for _, c := range cases {
		if got := oidcStripPrefix(c.path); got != c.want {
			t.Errorf("%s: oidcStripPrefix(%q) = %q, want %q", c.name, c.path, got, c.want)
		}
	}
}
