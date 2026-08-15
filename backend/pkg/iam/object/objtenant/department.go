package objtenant

type DepartmentBaseInfo struct {
	TenantID     string `json:"tenantID" form:"tenantID"`         // 租户ID
	ParentID     string `json:"parentID" form:"parentID"`         // 父部门ID
	Name         string `json:"name" form:"name"`                 // 部门名称
	Code         string `json:"code" form:"code"`                 // 部门编码
	Sort         int    `json:"sort" form:"sort"`                 // 排序
	LeaderUserID string `json:"leaderUserID" form:"leaderUserID"` // 部门负责人用户ID
}
