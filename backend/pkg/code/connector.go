package code

import "github.com/morehao/golib/gerror"

const (
	ConnectorCreateError      = 101000
	ConnectorDeleteError      = 101001
	ConnectorUpdateError      = 101002
	ConnectorGetDetailError   = 101003
	ConnectorGetPageListError = 101004
	ConnectorNotExistError    = 101005
)

var connectorErrorMsgMap = gerror.CodeMsgMap{
	ConnectorCreateError:      "创建连接器失败",
	ConnectorDeleteError:      "删除连接器失败",
	ConnectorUpdateError:      "修改连接器失败",
	ConnectorGetDetailError:   "查看连接器详情失败",
	ConnectorGetPageListError: "查看连接器列表失败",
	ConnectorNotExistError:    "连接器不存在",
}