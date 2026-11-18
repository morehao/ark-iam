package code

import "github.com/morehao/golib/gerror"

// 平台端身份连接器领域错误码。
// 连接器是平台级全局对象，管理面归 platformadmin（跨租户），故错误码归平台段（1009xx）。
// auth 运行时（Authorize/Callback 登录链路）仍沿用认证类错误码（code.AuthLoginFailedError 等）。
const (
	ConnectorCreateError      = 100920 // 创建连接器失败
	ConnectorDeleteError      = 100921 // 删除连接器失败
	ConnectorUpdateError      = 100922 // 修改连接器失败
	ConnectorGetDetailError   = 100923 // 查看连接器详情失败
	ConnectorGetPageListError = 100924 // 查看连接器列表失败
	ConnectorNotExistError    = 100925 // 连接器不存在

	// ConnectorDefaultTenantRequiredError 开启「自动创建用户」但未指定默认落租户/或指定的租户不存在。
	// 语义：自动开户必须能确定用户落哪个租户（default_tenant_id），否则拒绝。
	ConnectorDefaultTenantRequiredError = 100926 // 请指定有效的默认落租户
)

var connectorPlatformErrorMsgMap = gerror.CodeMsgMap{
	ConnectorCreateError:                "创建连接器失败",
	ConnectorDeleteError:                "删除连接器失败",
	ConnectorUpdateError:                "修改连接器失败",
	ConnectorGetDetailError:             "查看连接器详情失败",
	ConnectorGetPageListError:           "查看连接器列表失败",
	ConnectorNotExistError:              "连接器不存在",
	ConnectorDefaultTenantRequiredError: "请指定有效的默认落租户",
}

func init() {
	registerError(connectorPlatformErrorMsgMap)
}
