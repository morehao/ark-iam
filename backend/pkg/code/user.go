package code

import "github.com/morehao/golib/gerror"

const (
	UserCreateError      = 100500
	UserDeleteError      = 100501
	UserUpdateError      = 100502
	UserGetDetailError   = 100503
	UserGetPageListError = 100504
	UserNotExistError    = 100505
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
	UserDepartmentCreateError      = 104100
	UserDepartmentDeleteError      = 104101
	UserDepartmentUpdateError      = 104102
	UserDepartmentGetDetailError   = 104103
	UserDepartmentGetPageListError = 104104
	UserDepartmentNotExistError    = 104105
)

const (
	UserRoleCreateError      = 100750
	UserRoleDeleteError      = 100751
	UserRoleGetPageListError = 100752
	UserRoleNotExistError    = 100753
)

var userErrorMsgMap = gerror.CodeMsgMap{
	UserCreateError:      "创建用户失败",
	UserDeleteError:      "删除用户失败",
	UserUpdateError:      "修改用户失败",
	UserGetDetailError:   "查看用户详情失败",
	UserGetPageListError: "查看用户列表失败",
	UserNotExistError:    "用户不存在",
	UserIdentityCreateError:      "创建用户身份失败",
	UserIdentityDeleteError:      "删除用户身份失败",
	UserIdentityUpdateError:      "修改用户身份失败",
	UserIdentityGetDetailError:   "查看用户身份详情失败",
	UserIdentityGetPageListError: "查看用户身份列表失败",
	UserIdentityNotExistError:    "用户身份不存在",
	UserLoginLogGetDetailError:   "查看用户登录日志详情失败",
	UserLoginLogGetPageListError: "查看用户登录日志列表失败",
	UserLoginLogNotExistError:    "用户登录日志不存在",
	UserDepartmentCreateError:      "创建用户部门失败",
	UserDepartmentDeleteError:      "删除用户部门失败",
	UserDepartmentUpdateError:      "修改用户部门失败",
	UserDepartmentGetDetailError:   "查看用户部门详情失败",
	UserDepartmentGetPageListError: "查看用户部门列表失败",
	UserDepartmentNotExistError:    "用户部门不存在",
	UserRoleCreateError:      "创建用户角色关联失败",
	UserRoleDeleteError:      "删除用户角色关联失败",
	UserRoleGetPageListError: "查看用户角色关联列表失败",
	UserRoleNotExistError:   "用户角色关联不存在",
}