package dtotenant

import (
	"github.com/morehao/ark-iam/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateReq struct {
	objtenant.TenantBaseInfo
}

type TenantUpdateReq struct {
	TenantID uint `json:"tenantID" binding:"required"` // 租户ID
	objtenant.TenantBaseInfo
}

type TenantDetailReq struct {
	TenantID uint `json:"tenantID" form:"tenantID" binding:"required"` // 租户ID
}

type TenantPageListReq struct {
	gobject.PageQuery
}

type TenantDeleteReq struct {
	TenantID uint `json:"tenantID" binding:"required"` // 租户ID
}

type TenantCreateAsOwnerReq struct {
	PersonID uint   `json:"personID" binding:"required"` // 自然人ID
	Name     string `json:"name" binding:"required"`     // 租户名称
	AppID    uint   `json:"appID"`                       // 应用ID（可选，订阅该应用）
}

type DepartmentCreateReq struct {
	objtenant.DepartmentBaseInfo
}

type DepartmentUpdateReq struct {
	DepartmentID uint `json:"departmentID" binding:"required"` // 部门ID
	objtenant.DepartmentBaseInfo
}

type DepartmentDetailReq struct {
	DepartmentID uint `json:"departmentID" form:"departmentID" binding:"required"` // 部门ID
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
	TenantID uint `json:"tenantID" form:"tenantID"` // 租户ID
}
