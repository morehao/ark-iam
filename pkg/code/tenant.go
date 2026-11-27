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
	// P0 自定义域名/多 issuer 动作端点错误（见 tenant-custom-domain-redesign.md §5.3）。
	DomainFormatInvalidError    = 101207 // 域名格式非法
	DomainTenantSuspendedError  = 101208 // 租户不存在/挂起
	DomainReservedError         = 101209 // 域名保留/不可用
	DomainVerifyError           = 101210 // 验证失败
	DomainIssuerTierMismatchError = 101211 // issuer/tier 不匹配
	DomainStatusInvalidError    = 101212 // 状态/用途非法
	DomainRoleInvalidError      = 101213 // 角色非法
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
	DomainFormatInvalidError:            "域名格式非法",
	DomainTenantSuspendedError:          "租户不存在或已挂起",
	DomainReservedError:                 "域名保留或不可用",
	DomainVerifyError:                   "域名验证失败",
	DomainIssuerTierMismatchError:       "issuer 或 tier 不匹配",
	DomainStatusInvalidError:            "域名状态或用途非法",
	DomainRoleInvalidError:              "域名角色非法",
}
