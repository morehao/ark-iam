package dtoapplication

import "github.com/morehao/ark-iam/pkg/model"

type ApplicationCreateResp struct {
	AppID string `json:"appID"` // 应用ID
	Code  string `json:"code"`  // 应用编码
}

type ApplicationDetailResp struct {
	AppID                   string                   `json:"appID"`                   // 应用ID
	Code                    string                   `json:"code"`                    // 应用编码
	Name                    string                   `json:"name"`                    // 应用名称
	Description             string                   `json:"description"`             // 应用描述
	LogoURL                 string                   `json:"logoUrl"`                 // 应用logo
	HomepageURL             string                   `json:"homepageUrl"`             // 应用主页
	Source                  model.AppSource          `json:"source"`                  // 应用来源
	Status                  model.AppStatus          `json:"status"`                  // 状态
	Sort                    int                      `json:"sort"`                    // 排序
	AllowPersonCreateTenant *bool                    `json:"allowPersonCreateTenant"` // 个人是否可自助创建租户
	AllowJoinByInvite       *bool                    `json:"allowJoinByInvite"`       // 是否允许通过邀请加入租户
	RoleTemplate            []model.RoleTemplateItem `json:"roleTemplate"`            // 应用角色模板（跨系统授权契约值：code + 展示名）
	CreatedAt               int64                    `json:"createdAt"`               // 创建时间(unix 秒)
}

type PageListItem struct {
	AppID                   string                   `json:"appID"`                   // 应用ID
	Code                    string                   `json:"code"`                    // 应用编码
	Name                    string                   `json:"name"`                    // 应用名称
	Description             string                   `json:"description"`             // 应用描述
	Source                  model.AppSource          `json:"source"`                  // 应用来源
	Status                  model.AppStatus          `json:"status"`                  // 状态
	Sort                    int                      `json:"sort"`                    // 排序
	AllowPersonCreateTenant *bool                    `json:"allowPersonCreateTenant"` // 个人是否可自助创建租户
	AllowJoinByInvite       *bool                    `json:"allowJoinByInvite"`       // 是否允许通过邀请加入租户
	RoleTemplate            []model.RoleTemplateItem `json:"roleTemplate"`            // 应用角色模板（跨系统授权契约值：code + 展示名）
	CreatedAt               int64                    `json:"createdAt"`               // 创建时间(unix 秒)
	UpdatedAt               int64                    `json:"updatedAt"`               // 更新时间(unix 秒)
}

type ApplicationPageListResp struct {
	List  []PageListItem `json:"list"`  // 列表数据
	Total int64          `json:"total"` // 总数
}
