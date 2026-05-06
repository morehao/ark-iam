package code

import "github.com/morehao/golib/gerror"

const (
	LogGetDetailError   = 100800
	LogGetPageListError = 100801
	LogNotExistError    = 100802
)

var logErrorMsgMap = gerror.CodeMsgMap{
	LogGetDetailError:   "查看日志详情失败",
	LogGetPageListError: "查看日志列表失败",
	LogNotExistError:    "日志不存在",
}