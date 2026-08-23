package code

import "github.com/morehao/golib/gerror"

const (
	MenuCreateError      = 100600
	MenuDeleteError      = 100601
	MenuUpdateError      = 100602
	MenuGetDetailError   = 100603
	MenuGetPageListError = 100604
	MenuNotExistError    = 100605
)

const (
	RoleCreateError                 = 100700
	RoleDeleteError                 = 100701
	RoleUpdateError                 = 100702
	RoleGetDetailError              = 100703
	RoleGetPageListError            = 100704
	RoleNotExistError               = 100705
	RoleDeleteBuiltinForbiddenError = 100706
	RoleUpdateBuiltinForbiddenError = 100707
)

const (
	ApplicationCreateError         = 100730
	ApplicationDeleteError         = 100731
	ApplicationUpdateError         = 100732
	ApplicationGetDetailError      = 100733
	ApplicationGetPageListError    = 100734
	ApplicationNotExistError       = 100735
	ApplicationSystemBuiltInErr    = 100746
	ApplicationSecretCreateError   = 100736
	ApplicationSecretGetListError  = 100737
	ApplicationSecretDeleteError   = 100738
	ApplicationSecretNotExistError = 100739
)

const (
	ApplicationClientCreateError         = 100810
	ApplicationClientDeleteError         = 100811
	ApplicationClientUpdateError         = 100812
	ApplicationClientGetDetailError      = 100813
	ApplicationClientGetPageListError    = 100814
	ApplicationClientNotExistError       = 100815
	ApplicationClientSystemBuiltInErr    = 100820
	ApplicationClientSecretCreateError   = 100816
	ApplicationClientSecretGetListError  = 100817
	ApplicationClientSecretDeleteError   = 100818
	ApplicationClientSecretNotExistError = 100819
)

const (
	RoleMenuCreateError                   = 100760
	RoleMenuDeleteError                   = 100761
	RoleMenuGetPageListError              = 100762
	RoleMenuNotExistError                 = 100763
	RoleMenuAdminVisibilityForbiddenError = 100764 // 普通角色禁止授权管理员可见性（visibility=admin）菜单
)

const (
	RoleUserCreateError          = 100770
	RoleUserDeleteError          = 100771
	RoleUserGetListError         = 100772
	RoleUserNotExistError        = 100773
	RoleApplicationCreateError   = 100780
	RoleApplicationGetListError  = 100781
	RoleApplicationDeleteError   = 100782
	RoleApplicationNotExistError = 100783
)

var permissionErrorMsgMap = gerror.CodeMsgMap{
	MenuCreateError:                       "创建菜单失败",
	MenuDeleteError:                       "删除菜单失败",
	MenuUpdateError:                       "修改菜单失败",
	MenuGetDetailError:                    "查看菜单详情失败",
	MenuGetPageListError:                  "查看菜单列表失败",
	MenuNotExistError:                     "菜单不存在",
	RoleCreateError:                       "创建角色失败",
	RoleDeleteError:                       "删除角色失败",
	RoleUpdateError:                       "修改角色失败",
	RoleGetDetailError:                    "查看角色详情失败",
	RoleGetPageListError:                  "查看角色列表失败",
	RoleNotExistError:                     "角色不存在",
	RoleDeleteBuiltinForbiddenError:       "内置角色禁止删除",
	RoleUpdateBuiltinForbiddenError:       "内置角色禁止修改核心字段",
	ApplicationCreateError:                "创建应用失败",
	ApplicationDeleteError:                "删除应用失败",
	ApplicationUpdateError:                "修改应用失败",
	ApplicationGetDetailError:             "查看应用详情失败",
	ApplicationGetPageListError:           "查看应用列表失败",
	ApplicationNotExistError:              "应用不存在",
	ApplicationSystemBuiltInErr:           "应用为系统内置，不可删除",
	ApplicationSecretCreateError:          "创建应用密钥失败",
	ApplicationSecretGetListError:         "查看应用密钥列表失败",
	ApplicationSecretDeleteError:          "删除应用密钥失败",
	ApplicationSecretNotExistError:        "应用密钥不存在",
	ApplicationClientCreateError:          "创建OAuth客户端失败",
	ApplicationClientDeleteError:          "删除OAuth客户端失败",
	ApplicationClientUpdateError:          "修改OAuth客户端失败",
	ApplicationClientGetDetailError:       "查看OAuth客户端详情失败",
	ApplicationClientGetPageListError:     "查看OAuth客户端列表失败",
	ApplicationClientNotExistError:        "OAuth客户端不存在",
	ApplicationClientSystemBuiltInErr:     "OAuth客户端为系统内置，不可删除",
	ApplicationClientSecretCreateError:    "创建OAuth客户端密钥失败",
	ApplicationClientSecretGetListError:   "查看OAuth客户端密钥列表失败",
	ApplicationClientSecretDeleteError:    "删除OAuth客户端密钥失败",
	ApplicationClientSecretNotExistError:  "OAuth客户端密钥不存在",
	RoleMenuCreateError:                   "创建角色菜单关联失败",
	RoleMenuDeleteError:                   "删除角色菜单关联失败",
	RoleMenuGetPageListError:              "查看角色菜单关联列表失败",
	RoleMenuNotExistError:                 "角色菜单关联不存在",
	RoleMenuAdminVisibilityForbiddenError: "普通角色无法授权管理员可见性菜单",
	RoleUserCreateError:                   "创建角色用户关联失败",
	RoleUserDeleteError:                   "删除角色用户关联失败",
	RoleUserGetListError:                  "查看角色用户列表失败",
	RoleUserNotExistError:                 "角色用户不存在",
	RoleApplicationCreateError:            "创建角色应用关联失败",
	RoleApplicationGetListError:           "查看角色应用列表失败",
	RoleApplicationDeleteError:            "删除角色应用关联失败",
	RoleApplicationNotExistError:          "角色应用关联不存在",
}
