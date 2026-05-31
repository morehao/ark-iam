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
	OrganizationCreateError      = 100120
	OrganizationDeleteError      = 100121
	OrganizationUpdateError      = 100122
	OrganizationGetDetailError   = 100123
	OrganizationGetPageListError = 100124
	OrganizationNotExistError    = 100125
)

const (
	OrganizationRoleCreateError      = 100130
	OrganizationRoleDeleteError      = 100131
	OrganizationRoleUpdateError      = 100132
	OrganizationRoleGetDetailError   = 100133
	OrganizationRoleGetPageListError = 100134
	OrganizationRoleNotExistError    = 100135
)

const (
	OrganizationUserCreateError      = 100140
	OrganizationUserDeleteError      = 100141
	OrganizationUserGetPageListError = 100142
	OrganizationUserNotExistError    = 100143
)

const (
	OrganizationRoleUserCreateError      = 100150
	OrganizationRoleUserDeleteError      = 100151
	OrganizationRoleUserGetPageListError = 100152
	OrganizationRoleUserNotExistError    = 100153
)

const (
	DomainCreateError       = 101200
	DomainDeleteError       = 101201
	DomainGetPageListError  = 101202
	DomainNotExistError     = 101203
	DomainAlreadyExistError = 101204
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
	OrganizationUserCreateError:      "创建组织用户失败",
	OrganizationUserDeleteError:      "删除组织用户失败",
	OrganizationUserGetPageListError: "查看组织用户列表失败",
	OrganizationUserNotExistError:    "组织用户不存在",
	OrganizationRoleUserCreateError:      "创建组织角色用户失败",
	OrganizationRoleUserDeleteError:      "删除组织角色用户失败",
	OrganizationRoleUserGetPageListError: "查看组织角色用户列表失败",
	OrganizationRoleUserNotExistError:    "组织角色用户不存在",
	DomainCreateError:       "创建域名失败",
	DomainDeleteError:       "删除域名失败",
	DomainGetPageListError:  "查看域名列表失败",
	DomainNotExistError:     "域名不存在",
	DomainAlreadyExistError: "域名已存在",
}
