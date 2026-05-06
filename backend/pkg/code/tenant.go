package code

import "github.com/morehao/golib/gerror"

const (
	TenantCreateError      = 100100
	TenantDeleteError      = 100101
	TenantUpdateError      = 100102
	TenantGetDetailError   = 100103
	TenantGetPageListError = 100104
	TenantNotExistError    = 100105
)

var tenantErrorMsgMap = gerror.CodeMsgMap{
	TenantCreateError:      "创建租户管理失败",
	TenantDeleteError:      "删除租户管理失败",
	TenantUpdateError:      "修改租户管理失败",
	TenantGetDetailError:   "查看租户管理失败",
	TenantGetPageListError: "查看租户管理列表失败",
	TenantNotExistError:    "租户管理不存在",
}
