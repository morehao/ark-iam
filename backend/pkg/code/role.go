package code

import "github.com/morehao/golib/gerror"

const (
	RoleCreateError      = 100700
	RoleDeleteError      = 100701
	RoleUpdateError      = 100702
	RoleGetDetailError  = 100703
	RoleGetPageListError = 100704
	RoleNotExistError   = 100705
)

var roleErrorMsgMap = gerror.CodeMsgMap{
	RoleCreateError:      "创建角色失败",
	RoleDeleteError:      "删除角色失败",
	RoleUpdateError:      "修改角色失败",
	RoleGetDetailError:   "查看角色详情失败",
	RoleGetPageListError: "查看角色列表失败",
	RoleNotExistError:   "角色不存在",
}

const (
	ResourceCreateError      = 100710
	ResourceDeleteError      = 100711
	ResourceUpdateError      = 100712
	ResourceGetDetailError  = 100713
	ResourceGetPageListError = 100714
	ResourceNotExistError   = 100715
)

var resourceErrorMsgMap = gerror.CodeMsgMap{
	ResourceCreateError:      "创建资源失败",
	ResourceDeleteError:      "删除资源失败",
	ResourceUpdateError:      "修改资源失败",
	ResourceGetDetailError:   "查看资源详情失败",
	ResourceGetPageListError: "查看资源列表失败",
	ResourceNotExistError:   "资源不存在",
}

const (
	ScopeCreateError      = 100720
	ScopeDeleteError      = 100721
	ScopeUpdateError      = 100722
	ScopeGetDetailError  = 100723
	ScopeGetPageListError = 100724
	ScopeNotExistError   = 100725
)

var scopeErrorMsgMap = gerror.CodeMsgMap{
	ScopeCreateError:      "创建权限范围失败",
	ScopeDeleteError:      "删除权限范围失败",
	ScopeUpdateError:      "修改权限范围失败",
	ScopeGetDetailError:   "查看权限范围详情失败",
	ScopeGetPageListError: "查看权限范围列表失败",
	ScopeNotExistError:   "权限范围不存在",
}

const (
	ApplicationCreateError      = 100730
	ApplicationDeleteError      = 100731
	ApplicationUpdateError      = 100732
	ApplicationGetDetailError  = 100733
	ApplicationGetPageListError = 100734
	ApplicationNotExistError   = 100735
)

var applicationErrorMsgMap = gerror.CodeMsgMap{
	ApplicationCreateError:      "创建应用失败",
	ApplicationDeleteError:      "删除应用失败",
	ApplicationUpdateError:      "修改应用失败",
	ApplicationGetDetailError:   "查看应用详情失败",
	ApplicationGetPageListError: "查看应用列表失败",
	ApplicationNotExistError:   "应用不存在",
}

const (
	RoleScopeCreateError      = 100740
	RoleScopeDeleteError      = 100741
	RoleScopeGetPageListError = 100742
	RoleScopeNotExistError    = 100743
)

var roleScopeErrorMsgMap = gerror.CodeMsgMap{
	RoleScopeCreateError:      "创建角色权限关联失败",
	RoleScopeDeleteError:      "删除角色权限关联失败",
	RoleScopeGetPageListError: "查看角色权限关联列表失败",
	RoleScopeNotExistError:    "角色权限关联不存在",
}

const (
	UserRoleCreateError      = 100750
	UserRoleDeleteError      = 100751
	UserRoleGetPageListError = 100752
	UserRoleNotExistError    = 100753
)

var userRoleErrorMsgMap = gerror.CodeMsgMap{
	UserRoleCreateError:      "创建用户角色关联失败",
	UserRoleDeleteError:      "删除用户角色关联失败",
	UserRoleGetPageListError: "查看用户角色关联列表失败",
	UserRoleNotExistError:    "用户角色关联不存在",
}

const (
	RoleMenuCreateError      = 100760
	RoleMenuDeleteError      = 100761
	RoleMenuGetPageListError = 100762
	RoleMenuNotExistError    = 100763
)

var roleMenuErrorMsgMap = gerror.CodeMsgMap{
	RoleMenuCreateError:      "创建角色菜单关联失败",
	RoleMenuDeleteError:      "删除角色菜单关联失败",
	RoleMenuGetPageListError: "查看角色菜单关联列表失败",
	RoleMenuNotExistError:    "角色菜单关联不存在",
}

const (
	OrganizationCreateError      = 100770
	OrganizationDeleteError      = 100771
	OrganizationUpdateError      = 100772
	OrganizationGetDetailError  = 100773
	OrganizationGetPageListError = 100774
	OrganizationNotExistError   = 100775
)

var organizationErrorMsgMap = gerror.CodeMsgMap{
	OrganizationCreateError:      "创建组织失败",
	OrganizationDeleteError:      "删除组织失败",
	OrganizationUpdateError:      "修改组织失败",
	OrganizationGetDetailError:   "查看组织详情失败",
	OrganizationGetPageListError: "查看组织列表失败",
	OrganizationNotExistError:   "组织不存在",
}

const (
	OrganizationRoleCreateError      = 100780
	OrganizationRoleDeleteError      = 100781
	OrganizationRoleUpdateError      = 100782
	OrganizationRoleGetDetailError  = 100783
	OrganizationRoleGetPageListError = 100784
	OrganizationRoleNotExistError   = 100785
)

var organizationRoleErrorMsgMap = gerror.CodeMsgMap{
	OrganizationRoleCreateError:      "创建组织角色失败",
	OrganizationRoleDeleteError:      "删除组织角色失败",
	OrganizationRoleUpdateError:      "修改组织角色失败",
	OrganizationRoleGetDetailError:   "查看组织角色详情失败",
	OrganizationRoleGetPageListError: "查看组织角色列表失败",
	OrganizationRoleNotExistError:   "组织角色不存在",
}

const (
	OrganizationUserRelationCreateError      = 100790
	OrganizationUserRelationDeleteError      = 100791
	OrganizationUserRelationGetPageListError = 100792
	OrganizationUserRelationNotExistError    = 100793
)

var organizationUserRelationErrorMsgMap = gerror.CodeMsgMap{
	OrganizationUserRelationCreateError:      "创建组织用户关联失败",
	OrganizationUserRelationDeleteError:      "删除组织用户关联失败",
	OrganizationUserRelationGetPageListError: "查看组织用户关联列表失败",
	OrganizationUserRelationNotExistError:    "组织用户关联不存在",
}

const (
	OrganizationRoleUserRelationCreateError      = 100800
	OrganizationRoleUserRelationDeleteError      = 100801
	OrganizationRoleUserRelationGetPageListError = 100802
	OrganizationRoleUserRelationNotExistError    = 100803
)

var organizationRoleUserRelationErrorMsgMap = gerror.CodeMsgMap{
	OrganizationRoleUserRelationCreateError:      "创建组织角色用户关联失败",
	OrganizationRoleUserRelationDeleteError:      "删除组织角色用户关联失败",
	OrganizationRoleUserRelationGetPageListError: "查看组织角色用户关联列表失败",
	OrganizationRoleUserRelationNotExistError:    "组织角色用户关联不存在",
}