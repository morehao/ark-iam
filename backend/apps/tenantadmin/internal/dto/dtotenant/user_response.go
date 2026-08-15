package dtotenant

type UserPageListItem struct {
	UserID       string `json:"userID"`       // 用户ID
	TenantID     string `json:"tenantID"`     // 租户ID
	Username     string `json:"username"`     // 用户名
	PrimaryEmail string `json:"primaryEmail"` // 主要邮箱
	PrimaryPhone string `json:"primaryPhone"` // 主要手机号
	Name         string `json:"name"`         // 姓名
	Avatar       string `json:"avatar"`       // 头像URL
	IsSuspended bool   `json:"isSuspended"`  // 是否挂起
	CreatedAt    int64  `json:"createdAt"`    // 创建时间
}

type UserPageListResp struct {
	List  []UserPageListItem `json:"list"`  // 数据列表
	Total int64              `json:"total"` // 数据总条数
}
