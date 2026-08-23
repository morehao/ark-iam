package dtooidc

import "github.com/morehao/ark-iam/pkg/iam/object/objauth"

// RegisterPersonReq 注册 person 请求。OIDC 认证请求内完成 person 注册，
// 应用是否允许由 authRequestID 反查的 client→app 的 TenantPolicy 决定。
// Username/PrimaryEmail/PrimaryPhone 至少填写一个。
type RegisterPersonReq struct {
	AuthRequestID string `json:"authRequestID" binding:"required"`      // OIDC 授权票据ID（携带应用上下文）
	Username      string `json:"username" binding:"max=128"`            // 用户名（全局唯一标识，可选）
	PrimaryEmail  string `json:"primaryEmail" binding:"max=128"`        // 主要邮箱（全局唯一标识，可选）
	PrimaryPhone  string `json:"primaryPhone" binding:"max=32"`         // 主要手机号（全局唯一标识，可选）
	Password      string `json:"password" binding:"required,max=128"`   // 密码
	Name          string `json:"name" binding:"max=128"`                // 姓名
}

// RegisterPersonResp 注册 person 响应。
type RegisterPersonResp struct {
	PersonID              string                 `json:"personID"`                       // person ID
	RequiresPasswordLogin bool                   `json:"requiresPasswordLogin,omitempty"` // person 已存在需走密码登录
	RequiresTenantSelection bool                 `json:"requiresTenantSelection"`         // 是否需选租户（已注册且有租户）
	Tenants                 []objauth.TenantOption `json:"tenants,omitempty"`             // 该 person 已有租户列表
	AllowPersonCreateTenant bool                 `json:"allowPersonCreateTenant"`         // 应用允许 + 零租户 → 展示创建租户
}

// CreateTenantReq 创建租户请求（注册 person 后，为零租户 person 开通租户）。
type CreateTenantReq struct {
	AuthRequestID string `json:"authRequestID" binding:"required"`
	TenantName    string `json:"tenantName" binding:"required,max=128"`
	TenantCode    string `json:"tenantCode" binding:"max=64"`
}

// CreateTenantResp 创建租户响应。
type CreateTenantResp struct {
	TenantID string `json:"tenantID"`
	PersonID string `json:"personID"`
}
