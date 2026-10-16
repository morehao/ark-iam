package code

import "github.com/morehao/golib/gerror"

// audit 领域错误码段（1080XX，平台控制台审计只读查询）。
//
// 审计查询是**只读**接口：没有写入/更新/删除错误码；写入走 pkg/audit 的 best-effort 通道，
// 失败只记日志、不影响业务主流程，因此也不需要对外错误码。
const (
	// AuditLogGetPageListError 审计日志分页查询失败（系统错误）。
	AuditLogGetPageListError = 108001
	// AuditLogPageListParamInvalidError 审计日志查询入参非法（动作/结果不在枚举白名单内）。
	AuditLogPageListParamInvalidError = 108002
)

var auditErrorMsgMap = gerror.CodeMsgMap{
	AuditLogGetPageListError:          "审计日志查询失败",
	AuditLogPageListParamInvalidError: "审计日志查询条件不合法",
}
