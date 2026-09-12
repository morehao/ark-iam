package code

import "github.com/morehao/golib/gerror"

const (
	TenantCreateError                   = 100200
	TenantDeleteError                   = 100201
	TenantUpdateError                   = 100202
	TenantGetDetailError                = 100203
	TenantGetPageListError              = 100204
	TenantNotExistError                 = 100205
	TenantCreateAsOwnerForbiddenError   = 100206
	TenantSuspendedError                = 100207
	TenantSuspendSelfForbiddenError     = 100208
	TenantPageListStatusInvalidError    = 100209
	TenantAdminResetPasswordError       = 100210
	TenantPlatformSuspendForbiddenError = 100211 // 平台自运营租户不可挂起（被挂起会导致整栈控制台失联，且无恢复路径）
	TenantBuiltInDeleteForbiddenError   = 100212 // 平台自运营租户（种子租户 t_platform）不可删除（删除即整栈控制台失联，且无恢复路径）
)

const (
	DepartmentCreateError      = 100120
	DepartmentDeleteError      = 100121
	DepartmentUpdateError      = 100122
	DepartmentGetDetailError   = 100123
	DepartmentGetPageListError = 100124
	DepartmentNotExistError    = 100125
)

const (
	DepartmentUserCreateError         = 100140
	DepartmentUserDeleteError         = 100141
	DepartmentUserGetPageListError    = 100142
	DepartmentUserNotExistError       = 100143
	DepartmentUserUpdateError         = 100144
	DepartmentUserLeaderConflictError = 100145
)

const (
	DomainCreateError       = 101200
	DomainDeleteError       = 101201
	DomainGetPageListError  = 101202
	DomainNotExistError     = 101203
	DomainAlreadyExistError = 101204
	DomainUpdateError       = 101205
	DomainDetailError       = 101206
)

var tenantErrorMsgMap = gerror.CodeMsgMap{
	TenantCreateError:                   "创建租户管理失败",
	TenantDeleteError:                   "删除租户管理失败",
	TenantUpdateError:                   "修改租户管理失败",
	TenantGetDetailError:                "查看租户管理失败",
	TenantGetPageListError:              "查看租户管理列表失败",
	TenantNotExistError:                 "租户管理不存在",
	TenantCreateAsOwnerForbiddenError:   "当前自然人已拥有租户或应用策略禁止自助创建租户",
	TenantSuspendedError:                "该租户已被挂起",
	TenantSuspendSelfForbiddenError:     "不能挂起当前所在租户",
	TenantPlatformSuspendForbiddenError: "平台自运营租户不可挂起",
	TenantBuiltInDeleteForbiddenError:   "平台自运营租户由平台版本内置，不可删除",
	TenantPageListStatusInvalidError:    "租户状态筛选值不合法",
	TenantAdminResetPasswordError:       "重置租户内置管理员密码失败",
	DepartmentCreateError:               "创建部门失败",
	DepartmentDeleteError:               "删除部门失败",
	DepartmentUpdateError:               "修改部门失败",
	DepartmentGetDetailError:            "查看部门详情失败",
	DepartmentGetPageListError:          "查看部门列表失败",
	DepartmentNotExistError:             "部门不存在",
	DepartmentUserCreateError:           "创建部门成员失败",
	DepartmentUserDeleteError:           "删除部门成员失败",
	DepartmentUserGetPageListError:      "查看部门成员列表失败",
	DepartmentUserNotExistError:         "部门成员不存在",
	DepartmentUserUpdateError:           "修改部门成员失败",
	DepartmentUserLeaderConflictError:   "该部门已有负责人",
	DomainCreateError:                   "创建域名失败",
	DomainDeleteError:                   "删除域名失败",
	DomainGetPageListError:              "查看域名列表失败",
	DomainNotExistError:                 "域名不存在",
	DomainAlreadyExistError:             "域名已存在",
	DomainUpdateError:                   "更新域名失败",
	DomainDetailError:                   "查看域名详情失败",
}
