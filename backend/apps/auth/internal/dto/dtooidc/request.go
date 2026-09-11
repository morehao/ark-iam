package dtooidc

type OIDCLoginReq struct {
	AuthRequestID string `json:"authRequestID" binding:"required"`
	Identifier    string `json:"identifier" binding:"required"`
	Password      string `json:"password" binding:"required"`
}

// OIDCChangePasswordReq 首次登录强制改密请求：授权票据 + 当前（临时）密码 + 新密码。
// 不带 personID/tenantID：身份一律取自授权票据已绑定的 subject，避免参数污染。
type OIDCChangePasswordReq struct {
	AuthRequestID   string `json:"authRequestID" binding:"required"`
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required,max=128"`
}

type OIDCSelectTenantReq struct {
	AuthRequestID string `json:"authRequestID" binding:"required"`
	TenantID      string `json:"tenantID" binding:"required"`
}
