package code

import (
	"fmt"

	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/genericdao"
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

func GetError(code int) *gerror.Error {
	err := errorMap[code]
	return &err
}

func init() {
	// 业务错误码规范: 从 1001XX 开始
	// 领域划分: tenant(1001XX-1004XX) user(1005XX-1008XX) permission(1006XX-1009XX) auth(1010XX-1011XX) audit
	registerError(genericdao.DBErrorMsgMap)
	registerError(gconstant.SystemErrorMsgMap)
	registerError(gconstant.AuthErrorMsgMap)
	registerError(tenantErrorMsgMap)
	registerError(userErrorMsgMap)
	registerError(permissionErrorMsgMap)
	registerError(authErrorMsgMap)
	registerError(auditErrorMsgMap)
	registerError(oidcErrorMsgMap)
}