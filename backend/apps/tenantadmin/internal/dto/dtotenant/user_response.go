package dtotenant

type UserPageListItem struct {
	UserID         string `json:"userID"`         // 用户ID
	TenantID       string `json:"tenantID"`       // 租户ID
	Username       string `json:"username"`       // 用户名
	PrimaryEmail   string `json:"primaryEmail"`   // 主要邮箱
	PrimaryPhone   string `json:"primaryPhone"`   // 主要手机号
	Name           string `json:"name"`           // 姓名
	Avatar         string `json:"avatar"`         // 头像URL
	IsSuspended    bool   `json:"isSuspended"`    // 是否挂起
	PrimaryOrgName string `json:"primaryOrgName"` // 主组织名称
	RoleCount      int64  `json:"roleCount"`      // 角色数
	CreatedAt      int64  `json:"createdAt"`      // 创建时间
	UpdatedAt      int64  `json:"updatedAt"`      // 更新时间
}

type UserPageListResp struct {
	List  []UserPageListItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}

type UserCreateResp struct {
	UserID string `json:"userID"` // 用户ID
}

type UserDetailResp struct {
	UserPageListItem
	Organizations []UserOrganizationItem `json:"organizations"` // 组织归属
	Roles         []UserRoleItem         `json:"roles"`         // 已分配角色
}

type UserRoleItem struct {
	RoleID      string `json:"roleID"`      // 角色ID
	AppID       string `json:"appID"`       // 所属应用ID
	AppName     string `json:"appName"`     // 所属应用名称
	Name        string `json:"name"`        // 角色名称
	Code        string `json:"code"`        // 角色编码
	Description string `json:"description"` // 角色描述
}

type UserRolesListResp struct {
	List []UserRoleItem `json:"list"` // 角色列表
}

// ---------- 第三方身份（用户子资源） ----------

type UserIdentityItem struct {
	UserIdentityID string `json:"userIdentityID"` // 用户身份ID
	Issuer         string `json:"issuer"`         // 身份提供商
	IdentityID     string `json:"identityID"`     // 第三方用户ID
	Detail         any    `json:"detail"`         // 详细信息
	CreatedAt      int64  `json:"createdAt"`      // 创建时间(unix 秒)
	UpdatedAt      int64  `json:"updatedAt"`      // 更新时间(unix 秒)
}

type UserIdentityListResp struct {
	List  []UserIdentityItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}

type UserIdentityCreateResp struct {
	UserIdentityID string `json:"userIdentityID"` // 用户身份ID
}

// ---------- 登录日志（用户子资源） ----------

type UserLoginLogItem struct {
	UserLoginLogID string `json:"userLoginLogID"` // 登录日志ID
	LoginIP        string `json:"loginIP"`        // 登录IP地址
	UserAgent      string `json:"userAgent"`      // 用户代理信息
	LoginTime      int64  `json:"loginTime"`      // 登录时间(unix 秒)
}

type UserLoginLogListResp struct {
	List  []UserLoginLogItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}
