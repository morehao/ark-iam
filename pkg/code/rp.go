package code

import "github.com/morehao/golib/gerror"

// rp 领域错误码段（只读目录 API，/v1/rp/*）。
//
// 本应用只读、无写入路径，因此只有"查询"类错误码。
// 四个 HTTP 状态码语义在控制器层固定（设计文档关键决策六）：
// 401 无/坏令牌、403 缺 directory.read 或无 tenant_id、404 跨租户或不存在、503 目录不可用。
const (
	// RpDirectoryMembersError 成员批量/单个查询失败（系统错误）。
	RpDirectoryMembersError = 106000
	// RpDirectoryDepartmentError 部门树查询失败（系统错误）。
	RpDirectoryDepartmentError = 106001
	// RpDirectoryRolesError 角色清单查询失败（系统错误）。
	RpDirectoryRolesError = 106002
	// RpDirectoryNotFoundError 成员不存在或不属于调用方租户（二者不可区分）。
	RpDirectoryNotFoundError = 106003
	// RpDirectoryUnauthorizedError 无令牌/令牌不可用（HTTP 401）。
	RpDirectoryUnauthorizedError = 106004
	// RpDirectoryForbiddenError 缺 directory.read 或令牌无 tenant_id（HTTP 403）。
	RpDirectoryForbiddenError = 106005
	// RpDirectoryBadRequestError 入参非法（如批量 ids 超过上限，HTTP 400）。
	RpDirectoryBadRequestError = 106006
	// RpDirectoryUnavailableError 目录数据不可用（HTTP 503，SDK 侧 ErrDirectoryUnavailable）。
	RpDirectoryUnavailableError = 106007
)

var rpErrorMsgMap = gerror.CodeMsgMap{
	RpDirectoryMembersError:      "目录成员查询失败",
	RpDirectoryDepartmentError:   "部门树查询失败",
	RpDirectoryRolesError:        "角色清单查询失败",
	RpDirectoryNotFoundError:     "资源不存在",
	RpDirectoryUnauthorizedError: "未认证的请求",
	RpDirectoryForbiddenError:    "无权访问该资源",
	RpDirectoryBadRequestError:   "请求参数不合法",
	RpDirectoryUnavailableError:  "目录服务暂不可用",
}
