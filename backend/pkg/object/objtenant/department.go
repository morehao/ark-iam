package objtenant

import "github.com/morehao/ark-iam/pkg/model"

// DepartmentBaseInfo 部门节点基础信息（租户自服务部门树）。
type DepartmentBaseInfo struct {
	Name   string               `json:"name" form:"name" binding:"required"` // 部门名称
	Sort   int                  `json:"sort" form:"sort"`                    // 同级排序
	Status model.DeptNodeStatus `json:"status" form:"status"`                // 状态: enable-启用 disable-停用
}
