package dtoauth

type LoginReq struct {
	TenantID   uint   `json:"tenantID" binding:"required"`     // 租户ID
	Identifier string `json:"identifier" binding:"required"`   // 用户名/邮箱/手机号
	Password   string `json:"password" binding:"required"`     // 密码
}

type RegisterReq struct {
	TenantID     uint   `json:"tenantID" binding:"required"`       // 租户ID
	Username     string `json:"username" binding:"required"`         // 用户名
	PrimaryEmail string `json:"primaryEmail"`                       // 主要邮箱
	PrimaryPhone string `json:"primaryPhone"`                       // 主要手机号
	Password     string `json:"password" binding:"required"`       // 密码
	Name         string `json:"name"`                             // 姓名
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"` // 刷新令牌
}

type LogoutReq struct {
	RefreshToken string `json:"refreshToken"` // 刷新令牌
}

type UserinfoReq struct {
}

type SsoAuthorizationUrlReq struct {
	ConnectorID string `json:"connectorId" binding:"required"` // 连接器ID
}

type SsoCallbackReq struct {
	ConnectorID string `json:"connectorId" binding:"required"` // 连接器ID
	Code        string `json:"code" binding:"required"`       // 授权码
	State       string `json:"state"`                         // 状态
}

type AssignDepartmentsReq struct {
	UserID        uint   `json:"userID" binding:"required"`         // 用户ID
	DepartmentIDs []uint `json:"departmentIDs" binding:"required"`   // 部门ID列表
}