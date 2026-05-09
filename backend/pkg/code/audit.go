package code

import "github.com/morehao/golib/gerror"

const (
	LogGetDetailError   = 100900
	LogGetPageListError = 100901
	LogNotExistError    = 100902
)

var auditErrorMsgMap = gerror.CodeMsgMap{
	LogGetDetailError:   "查看日志详情失败",
	LogGetPageListError: "查看日志列表失败",
	LogNotExistError:    "日志不存在",
}
