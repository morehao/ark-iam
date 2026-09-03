package dtotenant

type UserPageListReq struct {
	Page           int    `json:"page" form:"page"`                     // 页码
	PageSize       int    `json:"pageSize" form:"pageSize"`             // 每页数量
	Keyword        string `json:"keyword" form:"keyword"`               // 关键词(姓名/用户名/邮箱/手机 模糊)
	IsSuspended    *bool  `json:"isSuspended" form:"isSuspended"`       // 状态过滤(挂起)
	OrganizationID string `json:"organizationID" form:"organizationID"` // 部门ID(仅筛选恰在该部门的用户,不含子部门)
}

type UserCreateReq struct {
	PersonID        string   `json:"personID"`                           // 已有自然人ID(可选,优先关联)
	Username        string   `json:"username"`                           // 全局用户名(可选)
	PrimaryEmail    string   `json:"primaryEmail"`                       // 主要邮箱
	PrimaryPhone    string   `json:"primaryPhone"`                       // 主要手机号
	Name            string   `json:"name" binding:"required"`            // 姓名(新建 person 时的自然人姓名)
	Avatar          string   `json:"avatar"`                             // 头像URL
	Password        string   `json:"password"`                           // 初始密码(可选,提供则 person 可登录)
	IsSuspended     bool     `json:"isSuspended"`                        // 是否挂起
	OrganizationIDs []string `json:"organizationIDs" binding:"required"` // 行政主部门ID列表(primary,至多1个,必传:用户必须从属部门)
	SecondaryOrgIDs []string `json:"secondaryOrgIDs"`                    // 参与部门ID列表(secondary,可多条,可选)
	LeaderOrgIDs    []string `json:"leaderOrgIDs"`                       // 负责部门ID列表(leader,可多条,可选;每部门至多1负责人)
}

type UserDetailReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserUpdateReq struct {
	UserID          string    `json:"-" uri:"userID" binding:"required"` // 用户ID
	Username        *string   `json:"username"`                          // 用户名(nil=不变)
	PrimaryEmail    *string   `json:"primaryEmail"`                      // 主要邮箱(nil=不变)
	PrimaryPhone    *string   `json:"primaryPhone"`                      // 主要手机号(nil=不变)
	Name            string    `json:"name"`                              // 姓名
	Avatar          string    `json:"avatar"`                            // 头像URL
	IsSuspended     *bool     `json:"isSuspended"`                       // 是否挂起
	PrimaryOrgID    *string   `json:"primaryOrgID"`                      // 主部门(primary,nil=不变;非nil=替换主部门,不可清空)
	SecondaryOrgIDs *[]string `json:"secondaryOrgIDs"`                   // 参与部门(secondary,nil=不变;[]=清空;含值=全量替换)
	LeaderOrgIDs    *[]string `json:"leaderOrgIDs"`                      // 负责部门(leader,nil=不变;[]=清空;含值=全量替换;每部门至多1负责人)
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
