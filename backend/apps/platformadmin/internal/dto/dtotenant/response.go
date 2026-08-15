package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateResp struct {
	TenantID string `json:"tenantID"` // 租户ID
}

type TenantCreateAsOwnerResp struct {
	TenantID string `json:"tenantID"` // 租户ID
}

type TenantDetailResp struct {
	TenantID string `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListItem struct {
	TenantID string `json:"tenantID"` // 租户ID
	objtenant.TenantBaseInfo
	gobject.OperatorBaseInfo
}

type TenantPageListResp struct {
	List  []TenantPageListItem `json:"list"`  // 数据列表
	Total int64                `json:"total"` // 数据总条数
}

type DepartmentCreateResp struct {
	DepartmentID string `json:"departmentID"` // 部门ID
}

type DepartmentDetailResp struct {
	DepartmentID string `json:"departmentID"` // 部门ID
	objtenant.DepartmentBaseInfo
	gobject.OperatorBaseInfo
}

type DepartmentPageListItem struct {
	DepartmentID string `json:"departmentID"` // 部门ID
	objtenant.DepartmentBaseInfo
	gobject.OperatorBaseInfo
}

type DepartmentPageListResp struct {
	List  []DepartmentPageListItem `json:"list"`  // 数据列表
	Total int64                    `json:"total"` // 数据总条数
}

type DepartmentTreeItem struct {
	DepartmentID string `json:"departmentID"` // 部门ID
	objtenant.DepartmentBaseInfo
	gobject.OperatorBaseInfo
	Children []DepartmentTreeItem `json:"children"` // 子部门
}

type DepartmentTreeResp struct {
	List []DepartmentTreeItem `json:"list"` // 部门树
}
