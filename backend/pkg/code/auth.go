package code

import "github.com/morehao/golib/gerror"

const (
	AuthIdentifierRequiredError = 110010
	AuthLoginFailedError       = 110011
	AuthRegisterFailedError    = 110012
)

const (
	UserSuspendedError          = 100506
	PasswordNotSetError        = 100507
	PasswordMismatchError      = 100508
	PasswordTooShortError      = 100509
	PasswordHashError         = 100510
	UsernameAlreadyExistsError = 100511
	EmailAlreadyExistsError    = 100512
	PhoneAlreadyExistsError    = 100513
	PasswordValidationError   = 100514
)

const (
	TokenGenerateError       = 110020
	RefreshTokenRequiredError = 110021
	RefreshTokenInvalidError = 110022
)

var authErrorMsgMap = gerror.CodeMsgMap{
	AuthIdentifierRequiredError: "用户标识不能为空",
	AuthLoginFailedError:       "登录失败",
	AuthRegisterFailedError:    "注册失败",
}

var userAuthErrorMsgMap = gerror.CodeMsgMap{
	UserSuspendedError:         "用户已被停用",
	PasswordNotSetError:        "密码未设置",
	PasswordMismatchError:       "密码错误",
	PasswordTooShortError:      "密码长度不足",
	PasswordHashError:          "密码加密失败",
	UsernameAlreadyExistsError: "用户名已存在",
	EmailAlreadyExistsError:    "邮箱已存在",
	PhoneAlreadyExistsError:    "手机号已存在",
	PasswordValidationError:    "密码不符合要求",
}

var tokenErrorMsgMap = gerror.CodeMsgMap{
	TokenGenerateError:       "生成令牌失败",
	RefreshTokenRequiredError: "刷新令牌不能为空",
	RefreshTokenInvalidError:  "刷新令牌无效",
}