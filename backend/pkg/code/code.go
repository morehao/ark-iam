package code

import (
	"fmt"

	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gerror"
)

var errorMap = gerror.ErrorMap{}

func registerError(codeMsgMap gerror.CodeMsgMap) {
	for code, msg := range codeMsgMap {
		if _, ok := errorMap[code]; ok {
			panic(fmt.Sprintf("error code %d already exists", code))
		}
		errorMap[code] = gerror.Error{
			Code: code,
			Msg:  msg,
		}
	}
}

// GetError 返回注册的业务错误哨兵值。
// golib/gerror 的哨兵设计为值类型（var ErrNotFound = Error{...}），
// errors.Is / errors.As 均按值匹配；若返回指针，每次调用都是新指针，
// 会导致 errors.Is(err, GetError(...)) 恒不成立，且 gincontext 无法
// 通过 errors.As 提取业务错误码。
func GetError(code int) gerror.Error {
	return errorMap[code]
}

func init() {
	// 业务错误码规范: 从 1001XX 开始
	// 领域划分: tenant(1001XX-1004XX) user(1005XX-1008XX) permission(1006XX-1009XX) auth(1010XX-1011XX) audit
	registerError(gconstant.DBErrorMsgMap)
	registerError(gconstant.SystemErrorMsgMap)
	registerError(gconstant.AuthErrorMsgMap)
	registerError(tenantErrorMsgMap)
	registerError(userErrorMsgMap)
	registerError(permissionErrorMsgMap)
	registerError(authErrorMsgMap)
	registerError(userAuthErrorMsgMap)
	registerError(connectorErrorMsgMap)
	registerError(auditErrorMsgMap)
	registerError(oidcErrorMsgMap)
}
