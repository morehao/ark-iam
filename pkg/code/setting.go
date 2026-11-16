package code

import "github.com/morehao/golib/gerror"

// 运行时配置中心（系统设置）领域错误码。
//
// 段位说明：1009XX 已被 API Key 占用、1012XX 已被占用，故此处取 1090XX。
const (
	SettingGroupNotAllowedError = 109000 // 不允许操作该配置分组
	SettingCreateError          = 109001 // 创建配置项失败
	SettingUpdateError          = 109002 // 更新配置项失败
	SettingDeleteError          = 109003 // 删除配置项失败
	SettingGetDetailError       = 109004 // 查看配置项详情失败
	SettingGetPageListError     = 109005 // 查看配置项列表失败
	SettingNotExistError        = 109006 // 配置项不存在
	SettingStatusInvalidError   = 109007 // 配置项状态非法
)

var settingErrorMsgMap = gerror.CodeMsgMap{
	SettingGroupNotAllowedError: "不允许操作该配置分组",
	SettingCreateError:          "创建配置项失败",
	SettingUpdateError:          "更新配置项失败",
	SettingDeleteError:          "删除配置项失败",
	SettingGetDetailError:       "查看配置项详情失败",
	SettingGetPageListError:     "查看配置项列表失败",
	SettingNotExistError:        "配置项不存在",
	SettingStatusInvalidError:   "配置项状态非法",
}
