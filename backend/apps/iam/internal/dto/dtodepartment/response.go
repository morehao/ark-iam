package dtodepartment

import (
	"github.com/morehao/ark-iam/iam/object/objdepartment"
	"github.com/morehao/golib/biz/gobject"
)

type DepartmentCreateResp struct {
	DepartmentID uint `json:"departmentID"` // 部门ID
}

type DepartmentDetailResp struct {
	DepartmentID uint `json:"departmentID"` // 部门ID
	objdepartment.DepartmentBaseInfo
	gobject.OperatorBaseInfo
}

type DepartmentPageListItem struct {
	DepartmentID uint `json:"departmentID"` // 部门ID
	objdepartment.DepartmentBaseInfo
	gobject.OperatorBaseInfo
}

type DepartmentPageListResp struct {
	List  []DepartmentPageListItem `json:"list"`  // 数据列表
	Total int64                   `json:"total"` // 数据总条数
}

type DepartmentTreeItem struct {
	DepartmentID uint `json:"departmentID"` // 部门ID
	objdepartment.DepartmentBaseInfo
	gobject.OperatorBaseInfo
	Children []DepartmentTreeItem `json:"children"` // 子部门
}

type DepartmentTreeResp struct {
	List []DepartmentTreeItem `json:"list"` // 部门树
}