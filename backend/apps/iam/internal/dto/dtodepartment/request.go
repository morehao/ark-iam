package dtodepartment

import (
	"github.com/morehao/ark-iam/iam/object/objdepartment"
	"github.com/morehao/golib/biz/gobject"
)

type DepartmentCreateReq struct {
	objdepartment.DepartmentBaseInfo
}

type DepartmentUpdateReq struct {
	DepartmentID uint `json:"departmentID" binding:"required"` // 部门ID
	objdepartment.DepartmentBaseInfo
}

type DepartmentDetailReq struct {
	DepartmentID uint `json:"departmentID" binding:"required"` // 部门ID
}

type DepartmentPageListReq struct {
	gobject.PageQuery
	TenantID   uint   `json:"tenantID"` // 租户ID
	ParentID   uint   `json:"parentID"` // 父部门ID
	Name       string `json:"name"`     // 部门名称
	Code       string `json:"code"`     // 部门编码
}

type DepartmentDeleteReq struct {
	DepartmentID uint `json:"departmentID" binding:"required"` // 部门ID
}

type DepartmentTreeReq struct {
	TenantID uint `json:"tenantID"` // 租户ID
}