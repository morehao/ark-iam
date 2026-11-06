package dtooidc

import "github.com/morehao/ark-iam/pkg/object/objidentity"

type OIDCLoginReq struct {
	AuthRequestID string `json:"authRequestID" binding:"required"`
	Identifier    string `json:"identifier" binding:"required"`
	Password      string `json:"password" binding:"required"`
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）：
	// 登录尚未鉴权，但 IP/UA 是登录日志与审计的必需元数据。
	Operator objidentity.Operator
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
