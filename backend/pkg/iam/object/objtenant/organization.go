package objtenant

// OrganizationBaseInfo 组织节点基础信息（租户自服务组织树）。
type OrganizationBaseInfo struct {
	Name   string `json:"name" form:"name" binding:"required"` // 组织名称
	Code   string `json:"code" form:"code"`                    // 组织编码(可空)
	Sort   int    `json:"sort" form:"sort"`                    // 同级排序
	Status string `json:"status" form:"status"`                // 状态: active-启用 inactive-停用
}
