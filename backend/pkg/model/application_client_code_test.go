package model

import "testing"

// TestClientCodePattern 客户端编码（= OIDC client_id）规则：小写字母开头，仅含小写字母与下划线。
// 比 AppCodePattern 更严——数字、连字符、大写、下划线开头、空串都不合法。
// 该规则同时约束控制台表单（前端 CLIENT_CODE_PATTERN）与 service 入口（IsValidClientCode），
// 两处口径必须一致。
func TestClientCodePattern(t *testing.T) {
	valid := []string{"platform_admin_web", "tenant_admin_web", "iam_client", "a", "my_client_web"}
	for _, code := range valid {
		if !IsValidClientCode(code) {
			t.Errorf("IsValidClientCode(%q) = false, want true", code)
		}
	}
	invalid := []string{
		"platform-admin-web", // 连字符
		"client1",            // 数字（本规则不允许，应用编码才允许）
		"cli_01a094d3",       // 数字
		"Platform_Admin",     // 大写
		"_client",            // 下划线开头
		"client web",         // 空格
		"client.web",         // 点号
		"",
	}
	for _, code := range invalid {
		if IsValidClientCode(code) {
			t.Errorf("IsValidClientCode(%q) = true, want false", code)
		}
	}
}

// TestSeedBuiltinClientCodesComply 内置客户端编码同时是网关侧的 audience 白名单值：
// 两个常量必须各自合规且互不相同——不合规会让该控制台令牌全部 401，
// 相同则两个控制台会互相接受对方签发的令牌。
func TestSeedBuiltinClientCodesComply(t *testing.T) {
	codes := []string{SeedBuiltinClientPlatformAdminWeb, SeedBuiltinClientTenantAdminWeb}
	seen := map[string]bool{}
	for _, code := range codes {
		if !IsValidClientCode(code) {
			t.Errorf("内置客户端编码 %q 不符合 ClientCodePattern", code)
		}
		if seen[code] {
			t.Errorf("内置客户端编码 %q 重复", code)
		}
		seen[code] = true
	}
}
