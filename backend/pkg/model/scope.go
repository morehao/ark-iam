package model

// OIDC 标准 scope 取值（禁止硬编码）。
//
// scope 是**开放集合**：RFC 6749 允许自定义 scope，op.Client.IsScopeAllowed 也接受任意值，
// 因此这里只固定协议标准值，不为元素引入具名类型、也不做闭合枚举——DefaultScopeList /
// ScopeList / ConnectorScopeList 的元素保持 string（见 jsontypes.go 的载具类型说明）。
// 常量声明为无类型字符串，可直接用于 []string 字面量、switch case 与上述载具类型。
const (
	ScopeOpenID  = "openid"  // OIDC Core 1.0：请求 ID token
	ScopeProfile = "profile" // OIDC Core 1.0：姓名/用户名等资料声明
	ScopeEmail   = "email"   // OIDC Core 1.0：邮箱与 email_verified
	ScopePhone   = "phone"   // OIDC Core 1.0：手机号与 phone_number_verified
)
