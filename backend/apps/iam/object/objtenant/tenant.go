package objtenant

type TenantBaseInfo struct {
	Code        string `json:"code" form:"code"`                 // 租户编码
	DbUser      string `json:"dbUser" form:"dbUser"`             // 数据库用户
	IsSuspended int8   `json:"isSuspended" form:"isSuspended"`   // 是否挂起
	Name        string `json:"name" form:"name"`                 // 租户名称
	Tag         string `json:"tag" form:"tag"`                   // 标签
}