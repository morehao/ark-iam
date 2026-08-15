package code

import "github.com/morehao/golib/gerror"

const (
	TenantCreateError                 = 100200
	TenantDeleteError                 = 100201
	TenantUpdateError                 = 100202
	TenantGetDetailError              = 100203
	TenantGetPageListError            = 100204
	TenantNotExistError               = 100205
	TenantCreateAsOwnerForbiddenError = 100206
)

const (
)

const (
	SystemCreateError      = 100300
	SystemDeleteError      = 100301
	SystemUpdateError      = 100302
	SystemGetDetailError   = 100303
	SystemGetPageListError = 100304
	SystemNotExistError    = 100305
)

const (
	OrganizationCreateError      = 100120
	OrganizationDeleteError      = 100121
	OrganizationUpdateError      = 100122
	OrganizationGetDetailError   = 100123
	OrganizationGetPageListError = 100124
	OrganizationNotExistError    = 100125
)

const (
	OrganizationUserCreateError      = 100140
	OrganizationUserDeleteError      = 100141
	OrganizationUserGetPageListError = 100142
	OrganizationUserNotExistError    = 100143
	OrganizationUserUpdateError      = 100144
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
	TenantCreateError:                    "创建租户管理失败",
	TenantDeleteError:                    "删除租户管理失败",
	TenantUpdateError:                    "修改租户管理失败",
	TenantGetDetailError:                 "查看租户管理失败",
	TenantGetPageListError:               "查看租户管理列表失败",
	TenantNotExistError:                  "租户管理不存在",
	TenantCreateAsOwnerForbiddenError:    "当前自然人已拥有租户或应用策略禁止自助创建租户",
	SystemCreateError:                    "创建系统配置失败",
	SystemDeleteError:                    "删除系统配置失败",
	SystemUpdateError:                    "修改系统配置失败",
	SystemGetDetailError:                 "查看系统配置失败",
	SystemGetPageListError:               "查看系统配置列表失败",
	SystemNotExistError:                  "系统配置不存在",
	OrganizationCreateError:              "创建组织失败",
	OrganizationDeleteError:              "删除组织失败",
	OrganizationUpdateError:              "修改组织失败",
	OrganizationGetDetailError:           "查看组织详情失败",
	OrganizationGetPageListError:         "查看组织列表失败",
	OrganizationNotExistError:            "组织不存在",
	OrganizationUserCreateError:          "创建组织用户失败",
	OrganizationUserDeleteError:          "删除组织用户失败",
	OrganizationUserGetPageListError:     "查看组织用户列表失败",
	OrganizationUserNotExistError:        "组织用户不存在",
	OrganizationUserUpdateError:          "修改组织用户失败",
	DomainCreateError:                    "创建域名失败",
	DomainDeleteError:                    "删除域名失败",
	DomainGetPageListError:               "查看域名列表失败",
	DomainNotExistError:                  "域名不存在",
	DomainAlreadyExistError:              "域名已存在",
	DomainUpdateError:                    "更新域名失败",
	DomainDetailError:                    "查看域名详情失败",
}
