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
			Code:	code,
			Msg:	msg,
		}
	}
}

func GetError(code int) *gerror.Error {
	err := errorMap[code]
	return &err
}

func init() {
	// 业务错误码规范: 从 1001XX 开始
	// 模块划分: 1001XX(组织) 1002XX(租户) 1003XX(系统) 1004XX(部门) 1005XX(用户) 1006XX(菜单) 1007XX(角色) 1008XX(日志) 1009XX(API密钥) 1010XX(连接器) 1011XX(SSO连接器) 1041XX(用户部门关系) 1051XX(用户身份)
	// 注: 100100-100109 被 application 使用
	registerError(genericdao.DBErrorMsgMap)
	registerError(gconstant.SystemErrorMsgMap)
	registerError(gconstant.AuthErrorMsgMap)
	registerError(tenantErrorMsgMap)
	registerError(systemErrorMsgMap)
	registerError(logErrorMsgMap)
	registerError(departmentErrorMsgMap)
	registerError(menuErrorMsgMap)
	registerError(connectorErrorMsgMap)
	registerError(ssoConnectorErrorMsgMap)
	registerError(userErrorMsgMap)
	registerError(userIdentityErrorMsgMap)
	registerError(userLoginLogErrorMsgMap)
	registerError(userDepartmentRelationErrorMsgMap)
	registerError(roleErrorMsgMap)
	registerError(resourceErrorMsgMap)
	registerError(scopeErrorMsgMap)
	registerError(applicationErrorMsgMap)
	registerError(roleScopeErrorMsgMap)
	registerError(userRoleErrorMsgMap)
	registerError(roleMenuErrorMsgMap)
	registerError(organizationErrorMsgMap)
	registerError(organizationRoleErrorMsgMap)
	registerError(organizationUserRelationErrorMsgMap)
	registerError(organizationRoleUserRelationErrorMsgMap)
	registerError(authErrorMsgMap)
	registerError(userAuthErrorMsgMap)
	registerError(tokenErrorMsgMap)
}
