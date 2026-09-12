package objtenant

// DepartmentBaseInfo 部门节点基础信息（租户自服务部门树）。
type DepartmentBaseInfo struct {
	Name   string `json:"name" form:"name" binding:"required"` // 部门名称
	Code   string `json:"code" form:"code"`                    // 部门编码(可空)
	Sort   int    `json:"sort" form:"sort"`                    // 同级排序
	Status string `json:"status" form:"status"`                // 状态: active-启用 inactive-停用
}
