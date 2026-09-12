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
}

// ApplicationUpdateReq 修改应用。source 不可改（内置应用不可被改写为第三方）。
type ApplicationUpdateReq struct {
	AppID                   string          `json:"-" uri:"appID" binding:"required"` // 应用ID
	Name                    string          `json:"name"`                             // 应用名称
	Description             string          `json:"description"`                      // 应用描述
	LogoURL                 string          `json:"logoUrl"`                          // 应用logo
	HomepageURL             string          `json:"homepageUrl"`                      // 应用主页
	Status                  model.AppStatus `json:"status"`                           // 状态: enable-启用, disable-停用
	Sort                    int             `json:"sort"`                             // 排序
	AllowPersonCreateTenant *bool           `json:"allowPersonCreateTenant"`          // 个人是否可自助创建租户
	AllowJoinByInvite       *bool           `json:"allowJoinByInvite"`                // 是否允许通过邀请加入租户
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
