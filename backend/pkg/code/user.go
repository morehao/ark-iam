package code

import "github.com/morehao/golib/gerror"

const (
	UserCreateError               = 100500
	UserDeleteError               = 100501
	UserUpdateError               = 100502
	UserGetDetailError            = 100503
	UserGetPageListError          = 100504
	UserNotExistError             = 100505
	UserOrganizationRequiredError = 100518 // 用户必须从属于至少一个部门
	UserAlreadyInTenantError      = 100516 // 自然人已在本租户内
	UserResetPasswordError        = 100517 // 重置密码失败
	UserContactRequiredError      = 100521 // 邮箱或手机号至少填写一个
	UserOwnerUpdateError          = 100522 // 指派/取消租户拥有者失败
)

const (
	UserIdentityCreateError      = 105100
	UserIdentityDeleteError      = 105101
	UserIdentityUpdateError      = 105102
	UserIdentityGetDetailError   = 105103
	UserIdentityGetPageListError = 105104
	UserIdentityNotExistError    = 105105
)

const (
	UserLoginLogGetDetailError   = 100800
	UserLoginLogGetPageListError = 100801
	UserLoginLogNotExistError    = 100802
)

const (
	UserRoleCreateError                   = 100750
	UserRoleDeleteError                   = 100751
	UserRoleGetPageListError              = 100752
	UserRoleNotExistError                 = 100753
	UserRoleReplaceError                  = 100754 // 全量替换用户角色失败
	UserRoleRemoveLastAdminForbiddenError = 100755 // 禁止移除最后一个内置管理员角色，防止系统管理能力锁死
)

// 服务账号（租户内机器主体，user_type=machine）领域错误码。
const (
	MachineUserCreateError        = 100830 // 创建服务账号失败
	MachineUserUpdateError        = 100831 // 修改服务账号失败
	MachineUserStatusUpdateError  = 100832 // 更新服务账号状态失败
	MachineUserDeleteError        = 100833 // 删除服务账号失败
	MachineUserGetPageListError   = 100834 // 查看服务账号列表失败
	MachineUserGetDetailError     = 100835 // 查看服务账号详情失败
	MachineUserRoleReplaceError   = 100836 // 更新服务账号角色失败
	UserSuperRoleAssignForbidden  = 100837 // 禁止将系统管理角色授予服务账号
	UserSystemAdminRequiredError  = 100838 // 需要系统管理能力(admin_level=super)
	MachineUserRoleGetListError   = 100839 // 查看服务账号角色失败
	MachineUserDeleteHasKeysError = 100840 // 删除服务账号前需先删除其全部API密钥
	UserMemberOperationOnlyError  = 100841 // 该操作仅支持对真实用户执行
)

// 租户端 API 密钥领域错误码（归属真实用户本人或服务账号）。
const (
	ApiKeyCreateError        = 100910 // 创建API密钥失败
	ApiKeyGetPageListError   = 100911 // 查看API密钥列表失败
	ApiKeyRevokeError        = 100912 // 吊销API密钥失败
	ApiKeyDeleteError        = 100913 // 删除API密钥失败
	ApiKeyNotExistError      = 100914 // API密钥不存在
	ApiKeyOwnerNotExistError = 100915 // API密钥归属用户不存在
	ApiKeyOwnerMismatchError = 100916 // API密钥归属用户与租户不匹配
)

var userErrorMsgMap = gerror.CodeMsgMap{
	UserCreateError:                       "创建用户失败",
	UserDeleteError:                       "删除用户失败",
	UserUpdateError:                       "修改用户失败",
	UserGetDetailError:                    "查看用户详情失败",
	UserGetPageListError:                  "查看用户列表失败",
	UserNotExistError:                     "用户不存在",
	UserOrganizationRequiredError:         "用户必须从属于至少一个部门",
	UserAlreadyInTenantError:              "该用户已在本租户内",
	UserResetPasswordError:                "重置密码失败",
	UserContactRequiredError:              "邮箱或手机号至少填写一个",
	UserOwnerUpdateError:                  "设置租户拥有者失败",
	UserIdentityCreateError:               "创建用户身份失败",
	UserIdentityDeleteError:               "删除用户身份失败",
	UserIdentityUpdateError:               "修改用户身份失败",
	UserIdentityGetDetailError:            "查看用户身份详情失败",
	UserIdentityGetPageListError:          "查看用户身份列表失败",
	UserIdentityNotExistError:             "用户身份不存在",
	UserLoginLogGetDetailError:            "查看用户登录日志详情失败",
	UserLoginLogGetPageListError:          "查看用户登录日志列表失败",
	UserLoginLogNotExistError:             "用户登录日志不存在",
	UserRoleCreateError:                   "创建用户角色关联失败",
	UserRoleDeleteError:                   "删除用户角色关联失败",
	UserRoleGetPageListError:              "查看用户角色关联列表失败",
	UserRoleNotExistError:                 "用户角色关联不存在",
	UserRoleReplaceError:                  "更新用户角色失败",
	UserRoleRemoveLastAdminForbiddenError: "禁止移除最后一个内置管理员角色，系统管理能力可能锁死",
	MachineUserCreateError:                "创建服务账号失败",
	MachineUserUpdateError:                "修改服务账号失败",
	MachineUserStatusUpdateError:          "更新服务账号状态失败",
	MachineUserDeleteError:                "删除服务账号失败",
	MachineUserGetPageListError:           "查看服务账号列表失败",
	MachineUserGetDetailError:             "查看服务账号详情失败",
	MachineUserRoleReplaceError:           "更新服务账号角色失败",
	MachineUserRoleGetListError:           "查看服务账号角色失败",
	UserSuperRoleAssignForbidden:          "禁止将系统管理角色授予服务账号",
	UserSystemAdminRequiredError:          "需要系统管理能力(admin_level=super)",
	MachineUserDeleteHasKeysError:         "请先删除该服务账号下的全部API密钥",
	UserMemberOperationOnlyError:          "该操作仅支持对真实用户执行",
	ApiKeyCreateError:                     "创建API密钥失败",
	ApiKeyGetPageListError:                "查看API密钥列表失败",
	ApiKeyRevokeError:                     "吊销API密钥失败",
	ApiKeyDeleteError:                     "删除API密钥失败",
	ApiKeyNotExistError:                   "API密钥不存在",
	ApiKeyOwnerNotExistError:              "API密钥归属用户不存在",
	ApiKeyOwnerMismatchError:              "API密钥归属用户与租户不匹配",
}
