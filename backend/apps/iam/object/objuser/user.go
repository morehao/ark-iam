package objuser

type UserBaseInfo struct {
	TenantID          uint   `json:"tenantID" form:"tenantID"`                     // 租户ID
	Username          string `json:"username" form:"username"`                       // 用户名
	PrimaryEmail      string `json:"primaryEmail" form:"primaryEmail"`               // 主要邮箱
	PrimaryPhone      string `json:"primaryPhone" form:"primaryPhone"`               // 主要手机号
	PasswordEncrypted string `json:"passwordEncrypted" form:"passwordEncrypted"`     // 加密密码
	PasswordMethod    string `json:"passwordMethod" form:"passwordMethod"`           // 密码加密方式
	Name              string `json:"name" form:"name"`                             // 姓名
	Avatar            string `json:"avatar" form:"avatar"`                         // 头像URL
	Profile           any    `json:"profile" form:"profile"`                       // 配置信息
	ApplicationID     uint   `json:"applicationID" form:"applicationID"`             // 应用ID
	Identities        any    `json:"identities" form:"identities"`                 // 第三方身份
	CustomData        any    `json:"customData" form:"customData"`                 // 自定义数据
	IsSuspended       int8   `json:"isSuspended" form:"isSuspended"`               // 是否挂起
}

type UserPasswordInfo struct {
	PasswordEncrypted string `json:"passwordEncrypted" form:"passwordEncrypted"` // 加密密码
	PasswordMethod    string `json:"passwordMethod" form:"passwordMethod"`       // 密码加密方式
}

type UserLoginInfo struct {
	LoginIP   string `json:"loginIP" form:"loginIP"`     // 登录IP地址
	UserAgent string `json:"userAgent" form:"userAgent"` // 用户代理信息
}

type UserIdentityBaseInfo struct {
	TenantID   uint   `json:"tenantID" form:"tenantID"`     // 租户ID
	UserID     uint   `json:"userID" form:"userID"`         // 用户ID
	Issuer     string `json:"issuer" form:"issuer"`         // 身份提供商
	IdentityID string `json:"identityID" form:"identityID"` // 第三方用户ID
	Detail     any    `json:"detail" form:"detail"`         // 详细信息
}

type UserLoginLogBaseInfo struct {
	TenantID  uint   `json:"tenantID" form:"tenantID"`   // 租户ID
	UserID    uint   `json:"userID" form:"userID"`       // 用户ID
	LoginIP   string `json:"loginIP" form:"loginIP"`     // 登录IP地址
	UserAgent string `json:"userAgent" form:"userAgent"` // 用户代理信息
	LoginTime int64  `json:"loginTime" form:"loginTime"` // 登录时间
}

type UserDepartmentRelationBaseInfo struct {
	TenantID     uint `json:"tenantID" form:"tenantID"`         // 租户ID
	UserID       uint `json:"userID" form:"userID"`             // 用户ID
	DepartmentID uint `json:"departmentID" form:"departmentID"` // 部门ID
	IsPrimary    int8 `json:"isPrimary" form:"isPrimary"`       // 是否主部门
}