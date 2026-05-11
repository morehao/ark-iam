package objauth

type LoginReq struct {
	Identifier string `json:"identifier" form:"identifier"` // 用户名/邮箱/手机号
	Password   string `json:"password" form:"password"`     // 密码
}

type RegisterReq struct {
	TenantID     uint   `json:"tenantID" form:"tenantID"`         // 租户ID
	Username     string `json:"username" form:"username"`           // 用户名
	PrimaryEmail string `json:"primaryEmail" form:"primaryEmail"`   // 主要邮箱
	PrimaryPhone string `json:"primaryPhone" form:"primaryPhone"`   // 主要手机号
	Password     string `json:"password" form:"password"`           // 密码
	Name         string `json:"name" form:"name"`                 // 姓名
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken" form:"refreshToken"` // 刷新令牌
}

type TokenInfo struct {
	AccessToken  string `json:"accessToken"`  // 访问令牌
	RefreshToken string `json:"refreshToken"` // 刷新令牌
	ExpiresIn    int64  `json:"expiresIn"`   // 过期时间(秒)
	TokenType    string `json:"tokenType"`    // 令牌类型
}

type PersonTokenInfo struct {
	TokenInfo
}

type TenantOption struct {
	TenantID uint   `json:"tenantID"` // 租户ID
	Name     string `json:"name"`     // 租户名称
	Tag      string `json:"tag"`      // 租户标识
	UserID   uint   `json:"userID"`   // 用户ID
	IsOwner  int8   `json:"isOwner"`  // 是否租户拥有者
}

type UserInfo struct {
	UserID       uint   `json:"userID"`        // 用户ID
	TenantID     uint   `json:"tenantID"`      // 租户ID
	Username     string `json:"username"`      // 用户名
	PrimaryEmail string `json:"primaryEmail"`   // 主要邮箱
	PrimaryPhone string `json:"primaryPhone"`   // 主要手机号
	Name         string `json:"name"`          // 姓名
	Avatar       string `json:"avatar"`        // 头像
}

type PersonInfo struct {
	PersonID uint   `json:"personID"` // 自然人ID
	Name     string `json:"name"`     // 姓名
	Avatar   string `json:"avatar"`   // 头像
}

type TenantUserInfo struct {
	UserID   uint   `json:"userID"`   // 用户ID
	TenantID uint   `json:"tenantID"` // 租户ID
	Name     string `json:"name"`     // 姓名
	IsOwner  int8   `json:"isOwner"`  // 是否租户拥有者
}
