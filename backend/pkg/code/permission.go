package code

import "github.com/morehao/golib/gerror"

const (
	MenuCreateError      = 100600
	MenuDeleteError      = 100601
	MenuUpdateError      = 100602
	MenuGetDetailError   = 100603
	MenuGetPageListError = 100604
	MenuNotExistError    = 100605
)

const (
	RoleCreateError      = 100700
	RoleDeleteError      = 100701
	RoleUpdateError      = 100702
	RoleGetDetailError   = 100703
	RoleGetPageListError = 100704
	RoleNotExistError    = 100705
)

const (
	ResourceCreateError      = 100710
	ResourceDeleteError      = 100711
	ResourceUpdateError      = 100712
	ResourceGetDetailError   = 100713
	ResourceGetPageListError = 100714
	ResourceNotExistError    = 100715
)

const (
	ScopeCreateError      = 100720
	ScopeDeleteError      = 100721
	ScopeUpdateError      = 100722
	ScopeGetDetailError   = 100723
	ScopeGetPageListError = 100724
	ScopeNotExistError    = 100725
)

const (
	ApplicationCreateError         = 100730
	ApplicationDeleteError         = 100731
	ApplicationUpdateError         = 100732
	ApplicationGetDetailError      = 100733
	ApplicationGetPageListError    = 100734
	ApplicationNotExistError       = 100735
	ApplicationSecretCreateError   = 100736
	ApplicationSecretGetListError  = 100737
	ApplicationSecretDeleteError   = 100738
	ApplicationSecretNotExistError = 100739
)

const (
	RoleMenuCreateError      = 100760
	RoleMenuDeleteError      = 100761
	RoleMenuGetPageListError = 100762
	RoleMenuNotExistError    = 100763
)

const (
	RoleScopeCreateError      = 100740
	RoleScopeDeleteError      = 100741
	RoleScopeGetPageListError = 100742
	RoleScopeNotExistError    = 100743
)

const (
	RoleUserCreateError         = 100770
	RoleUserDeleteError         = 100771
	RoleUserGetListError        = 100772
	RoleUserNotExistError       = 100773
	RoleApplicationCreateError  = 100780
	RoleApplicationGetListError = 100781
	RoleApplicationDeleteError  = 100782
	RoleApplicationNotExistError = 100783
)

var permissionErrorMsgMap = gerror.CodeMsgMap{
	MenuCreateError:      "创建菜单失败",
	MenuDeleteError:      "删除菜单失败",
	MenuUpdateError:      "修改菜单失败",
	MenuGetDetailError:   "查看菜单详情失败",
	MenuGetPageListError: "查看菜单列表失败",
	MenuNotExistError:    "菜单不存在",
	RoleCreateError:      "创建角色失败",
	RoleDeleteError:      "删除角色失败",
	RoleUpdateError:      "修改角色失败",
	RoleGetDetailError:   "查看角色详情失败",
	RoleGetPageListError: "查看角色列表失败",
	RoleNotExistError:    "角色不存在",
	ResourceCreateError:      "创建资源失败",
	ResourceDeleteError:      "删除资源失败",
	ResourceUpdateError:      "修改资源失败",
	ResourceGetDetailError:   "查看资源详情失败",
	ResourceGetPageListError: "查看资源列表失败",
	ResourceNotExistError:    "资源不存在",
	ScopeCreateError:      "创建权限范围失败",
	ScopeDeleteError:      "删除权限范围失败",
	ScopeUpdateError:      "修改权限范围失败",
	ScopeGetDetailError:   "查看权限范围详情失败",
	ScopeGetPageListError: "查看权限范围列表失败",
	ScopeNotExistError:   "权限范围不存在",
	ApplicationCreateError:         "创建应用失败",
	ApplicationDeleteError:         "删除应用失败",
	ApplicationUpdateError:         "修改应用失败",
	ApplicationGetDetailError:      "查看应用详情失败",
	ApplicationGetPageListError:    "查看应用列表失败",
	ApplicationNotExistError:       "应用不存在",
	ApplicationSecretCreateError:   "创建应用密钥失败",
	ApplicationSecretGetListError:  "查看应用密钥列表失败",
	ApplicationSecretDeleteError:   "删除应用密钥失败",
	ApplicationSecretNotExistError: "应用密钥不存在",
	RoleMenuCreateError:      "创建角色菜单关联失败",
	RoleMenuDeleteError:      "删除角色菜单关联失败",
	RoleMenuGetPageListError: "查看角色菜单关联列表失败",
	RoleMenuNotExistError:    "角色菜单关联不存在",
	RoleScopeCreateError:      "创建角色权限关联失败",
	RoleScopeDeleteError:      "删除角色权限关联失败",
	RoleScopeGetPageListError: "查看角色权限关联列表失败",
	RoleScopeNotExistError:    "角色权限关联不存在",
	RoleUserCreateError:         "创建角色用户关联失败",
	RoleUserDeleteError:         "删除角色用户关联失败",
	RoleUserGetListError:        "查看角色用户列表失败",
	RoleUserNotExistError:       "角色用户不存在",
	RoleApplicationCreateError:  "创建角色应用关联失败",
	RoleApplicationGetListError: "查看角色应用列表失败",
	RoleApplicationDeleteError:  "删除角色应用关联失败",
	RoleApplicationNotExistError: "角色应用关联不存在",
}