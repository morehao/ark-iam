package code

import "github.com/morehao/golib/gerror"

const (
	UserCreateError         = 100500
	UserDeleteError         = 100501
	UserUpdateError         = 100502
	UserGetDetailError      = 100503
	UserGetPageListError    = 100504
	UserNotExistError       = 100505
	PersonCreateError       = 100506
	UsernameDuplicateError  = 100507
	PasswordComplexityError = 100508
)

var userErrorMsgMap = gerror.CodeMsgMap{
	UserCreateError:         "创建用户管理失败",
	UserDeleteError:         "删除用户管理失败",
	UserUpdateError:         "修改用户管理失败",
	UserGetDetailError:      "查看用户管理失败",
	UserGetPageListError:    "查看用户管理失败",
	UserNotExistError:       "用户管理不存在",
	PersonCreateError:       "创建自然人失败",
	UsernameDuplicateError:  "用户名已存在",
	PasswordComplexityError: "密码复杂度不符合要求",
}
