package dtotenant

type UserPageListReq struct {
	Page        int    `json:"page" form:"page"`                  // 页码
	PageSize    int    `json:"pageSize" form:"pageSize"`          // 每页数量
	Keyword     string `json:"keyword" form:"keyword"`            // 关键词(姓名/用户名/邮箱/手机 模糊)
	IsSuspended *bool  `json:"isSuspended" form:"isSuspended"`    // 状态过滤(挂起)
}

type UserCreateReq struct {
	PersonID        string   `json:"personID"`         // 已有自然人ID(可选,优先关联)
	Username        string   `json:"username"`         // 全局用户名(可选)
	PrimaryEmail    string   `json:"primaryEmail"`     // 主要邮箱
	PrimaryPhone    string   `json:"primaryPhone"`     // 主要手机号
	Name            string   `json:"name" binding:"required"` // 姓名(新建 person 时的自然人姓名)
	Avatar          string   `json:"avatar"`           // 头像URL
	Password        string   `json:"password"`         // 初始密码(可选,提供则 person 可登录)
	IsSuspended     bool     `json:"isSuspended"`      // 是否挂起
	OrganizationIDs []string `json:"organizationIDs" binding:"required"` // 归属组织ID列表(首个为主组织,必传:用户必须从属于部门)
}

type UserDetailReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserUpdateReq struct {
	UserID      string `json:"-" uri:"userID" binding:"required"` // 用户ID
	Name        string `json:"name"`                              // 姓名
	Avatar      string `json:"avatar"`                            // 头像URL
	IsSuspended *bool  `json:"isSuspended"`                       // 是否挂起
}

type UserResetPasswordReq struct {
	UserID   string `json:"-" uri:"userID" binding:"required"` // 用户ID
	Password string `json:"password" binding:"required"`       // 新密码
}

type UserRolesListReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserRolesUpdateReq struct {
	UserID  string   `json:"-" uri:"userID" binding:"required"` // 用户ID
	RoleIDs []string `json:"roleIDs" binding:"required"`        // 角色ID列表(全量替换)
}
