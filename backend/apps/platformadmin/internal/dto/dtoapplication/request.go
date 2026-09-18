package dtoapplication

import "github.com/morehao/ark-iam/pkg/model"

// ApplicationCreateReq 创建应用。
// 来源（source）不由控制台决定：控制台创建的应用恒为 model.AppSourceThirdParty，
// builtin / first_party 只能由种子与运维产生。
type ApplicationCreateReq struct {
	Code                    string `json:"code" binding:"required"` // 应用编码：下划线连接（小写字母开头，仅含小写字母、数字与下划线，见 model.AppCodePattern）
	Name                    string `json:"name" binding:"required"` // 应用名称
	Description             string `json:"description"`             // 应用描述
	LogoURL                 string `json:"logoUrl"`                 // 应用logo
	HomepageURL             string `json:"homepageUrl"`             // 应用主页
	Sort                    int    `json:"sort"`                    // 排序
	AllowPersonCreateTenant *bool  `json:"allowPersonCreateTenant"` // 个人是否可自助创建租户
	AllowJoinByInvite       *bool  `json:"allowJoinByInvite"`       // 是否允许通过邀请加入租户
	// RoleTemplate 应用角色模板：本应用对外提供的跨系统授权契约值（code + 展示名）。
	// 下游按「claim_prefix + code」认策略名（策略在下游是全局命名实体、全租户共用一条），
	// 故契约值由应用方在此定义一次，租户只能授权、不能定义；开通应用时物化到各租户。
	RoleTemplate []model.RoleTemplateItem `json:"roleTemplate"`
}

// ApplicationUpdateReq 修改应用。source 不可改（内置应用不可被改写为第三方）。
type ApplicationUpdateReq struct {
	AppID string `json:"-" uri:"appID" binding:"required"` // 应用ID
	// Code 应用编码：自建应用可改（留空表示不修改），规则同创建（model.AppCodePattern，小写字母开头、
	// 仅小写字母/数字/下划线）；改它不影响菜单/订阅/角色——那些都挂 app_id。
	// **内置应用**（source=builtin）拒改——控制台菜单入口仍按该编码定位（platform_admin/tenant_admin），
	// 改名会当场让对应控制台侧边栏失联且无法从界面恢复。
	Code                    string          `json:"code"`                    // 应用编码（留空不修改）
	Name                    string          `json:"name"`                    // 应用名称
	Description             string          `json:"description"`             // 应用描述
	LogoURL                 string          `json:"logoUrl"`                 // 应用logo
	HomepageURL             string          `json:"homepageUrl"`             // 应用主页
	Status                  model.AppStatus `json:"status"`                  // 状态: enable-启用, disable-停用
	Sort                    int             `json:"sort"`                    // 排序
	AllowPersonCreateTenant *bool           `json:"allowPersonCreateTenant"` // 个人是否可自助创建租户
	AllowJoinByInvite       *bool           `json:"allowJoinByInvite"`       // 是否允许通过邀请加入租户
	// RoleTemplate 应用角色模板：传 null 表示不修改，传 [] 表示清空，传值即全量替换。
	// 全量替换会**撤下**已不在模板中的模板角色（连带清理其用户/菜单授权），界面需二次确认。
	RoleTemplate []model.RoleTemplateItem `json:"roleTemplate"`
}

type ApplicationDetailReq struct {
	AppID string `json:"-" uri:"appID" binding:"required"` // 应用ID
}

type ApplicationDeleteReq struct {
	AppID string `json:"-" uri:"appID" binding:"required"` // 应用ID
}

type ApplicationPageListReq struct {
	Page     int             `json:"page" form:"page"`         // 页码
	PageSize int             `json:"pageSize" form:"pageSize"` // 每页条数
	Name     string          `json:"name" form:"name"`         // 应用名称（模糊搜索）
	Source   model.AppSource `json:"source" form:"source"`     // 应用来源: builtin-内置, first_party-第一方, third_party-第三方
	Status   model.AppStatus `json:"status" form:"status"`     // 状态
}
