package code

import "github.com/morehao/golib/gerror"

const (
	SsoConnectorCreateError      = 101100
	SsoConnectorDeleteError      = 101101
	SsoConnectorUpdateError      = 101102
	SsoConnectorGetDetailError   = 101103
	SsoConnectorGetPageListError = 101104
	SsoConnectorNotExistError    = 101105
	SsoAuthFailedError           = 101106
)

var ssoConnectorErrorMsgMap = gerror.CodeMsgMap{
	SsoConnectorCreateError:      "创建SSO连接器失败",
	SsoConnectorDeleteError:      "删除SSO连接器失败",
	SsoConnectorUpdateError:      "修改SSO连接器失败",
	SsoConnectorGetDetailError:   "查看SSO连接器详情失败",
	SsoConnectorGetPageListError: "查看SSO连接器列表失败",
	SsoConnectorNotExistError:    "SSO连接器不存在",
	SsoAuthFailedError:           "SSO认证失败",
}