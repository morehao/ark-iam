package code

import "github.com/morehao/golib/gerror"

const (
	SystemCreateError      = 100300
	SystemDeleteError      = 100301
	SystemUpdateError      = 100302
	SystemGetDetailError   = 100303
	SystemGetPageListError = 100304
	SystemNotExistError    = 100305
)

var systemErrorMsgMap = gerror.CodeMsgMap{
	SystemCreateError:      "创建系统配置失败",
	SystemDeleteError:      "删除系统配置失败",
	SystemUpdateError:      "修改系统配置失败",
	SystemGetDetailError:   "查看系统配置失败",
	SystemGetPageListError: "查看系统配置列表失败",
	SystemNotExistError:    "系统配置不存在",
}