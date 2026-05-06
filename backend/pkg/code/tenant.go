package code

import "github.com/morehao/golib/gerror"

const (
	TenantCreateError      = 100200
	TenantDeleteError      = 100201
	TenantUpdateError      = 100202
	TenantGetDetailError   = 100203
	TenantGetPageListError = 100204
	TenantNotExistError    = 100205
)

const (
	DepartmentCreateError      = 100400
	DepartmentDeleteError      = 100401
	DepartmentUpdateError      = 100402
	DepartmentGetDetailError   = 100403
	DepartmentGetPageListError = 100404
	DepartmentNotExistError    = 100405
)

const (
	SystemCreateError      = 100300
	SystemDeleteError      = 100301
	SystemUpdateError      = 100302
	SystemGetDetailError   = 100303
	SystemGetPageListError = 100304
	SystemNotExistError    = 100305
)

const (
	OrganizationCreateError      = 100100
	OrganizationDeleteError      = 100101
	OrganizationUpdateError      = 100102
	OrganizationGetDetailError   = 100103
	OrganizationGetPageListError = 100104
	OrganizationNotExistError    = 100105
)

const (
	OrganizationRoleCreateError      = 100780
	OrganizationRoleDeleteError      = 100781
	OrganizationRoleUpdateError      = 100782
	OrganizationRoleGetDetailError   = 100783
	OrganizationRoleGetPageListError = 100784
	OrganizationRoleNotExistError    = 100785
)

const (
	OrganizationUserRelationCreateError      = 100790
	OrganizationUserRelationDeleteError      = 100791
	OrganizationUserRelationGetPageListError = 100792
	OrganizationUserRelationNotExistError    = 100793
)

const (
	OrganizationRoleUserRelationCreateError      = 100800
	OrganizationRoleUserRelationDeleteError      = 100801
	OrganizationRoleUserRelationGetPageListError = 100802
	OrganizationRoleUserRelationNotExistError    = 100803
)

var tenantErrorMsgMap = gerror.CodeMsgMap{
	TenantCreateError:      "创建租户管理失败",
	TenantDeleteError:      "删除租户管理失败",
	TenantUpdateError:      "修改租户管理失败",
	TenantGetDetailError:   "查看租户管理失败",
	TenantGetPageListError: "查看租户管理列表失败",
	TenantNotExistError:    "租户管理不存在",
	DepartmentCreateError:      "创建部门失败",
	DepartmentDeleteError:      "删除部门失败",
	DepartmentUpdateError:      "修改部门失败",
	DepartmentGetDetailError:   "查看部门详情失败",
	DepartmentGetPageListError: "查看部门列表失败",
	DepartmentNotExistError:    "部门不存在",
	SystemCreateError:      "创建系统配置失败",
	SystemDeleteError:      "删除系统配置失败",
	SystemUpdateError:      "修改系统配置失败",
	SystemGetDetailError:   "查看系统配置失败",
	SystemGetPageListError: "查看系统配置列表失败",
	SystemNotExistError:    "系统配置不存在",
	OrganizationCreateError:      "创建组织失败",
	OrganizationDeleteError:      "删除组织失败",
	OrganizationUpdateError:      "修改组织失败",
	OrganizationGetDetailError:   "查看组织详情失败",
	OrganizationGetPageListError: "查看组织列表失败",
	OrganizationNotExistError:   "组织不存在",
	OrganizationRoleCreateError:      "创建组织角色失败",
	OrganizationRoleDeleteError:      "删除组织角色失败",
	OrganizationRoleUpdateError:      "修改组织角色失败",
	OrganizationRoleGetDetailError:   "查看组织角色详情失败",
	OrganizationRoleGetPageListError: "查看组织角色列表失败",
	OrganizationRoleNotExistError:   "组织角色不存在",
	OrganizationUserRelationCreateError:      "创建组织用户关联失败",
	OrganizationUserRelationDeleteError:      "删除组织用户关联失败",
	OrganizationUserRelationGetPageListError: "查看组织用户关联列表失败",
	OrganizationUserRelationNotExistError:    "组织用户关联不存在",
	OrganizationRoleUserRelationCreateError:      "创建组织角色用户关联失败",
	OrganizationRoleUserRelationDeleteError:      "删除组织角色用户关联失败",
	OrganizationRoleUserRelationGetPageListError: "查看组织角色用户关联列表失败",
	OrganizationRoleUserRelationNotExistError:    "组织角色用户关联不存在",
}