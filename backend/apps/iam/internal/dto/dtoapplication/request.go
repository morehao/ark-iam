package dtoapplication

type CreateReq struct {
	Code        string `json:"code" binding:"required"`        // 应用编码
	Name        string `json:"name" binding:"required"`        // 应用名称
	Description string `json:"description"`                    // 应用描述
	LogoURL     string `json:"logoUrl"`                        // 应用logo
	HomepageURL string `json:"homepageUrl"`                    // 应用主页
	Type        string `json:"type"`                           // 应用类型: first_party-第一方, third_party-第三方
	Visibility  string `json:"visibility" binding:"required"`  // 可见性: public-所有租户, private-仅平台租户
	Sort        int    `json:"sort"`                           // 排序
}

type UpdateReq struct {
	AppID    uint   `json:"appId" binding:"required"`    // 应用ID
	Name        string `json:"name"`                           // 应用名称
	Description string `json:"description"`                    // 应用描述
	LogoURL     string `json:"logoUrl"`                        // 应用logo
	HomepageURL string `json:"homepageUrl"`                    // 应用主页
	Type        string `json:"type"`                           // 应用类型: first_party-第一方, third_party-第三方
	Visibility  string `json:"visibility"`                     // 可见性: public-所有租户, private-仅平台租户
	Status      string `json:"status"`                         // 状态: enable-启用, disable-停用
	Sort        int    `json:"sort"`                           // 排序
}

type DetailReq struct {
	AppID uint `form:"appId" binding:"required"`         // 应用ID
}

type DeleteReq struct {
	AppID uint `json:"appId" binding:"required"`         // 应用ID
}

type PageListReq struct {
	Page     int    `json:"page"`                              // 页码
	PageSize int    `json:"pageSize"`                          // 每页条数
	Name     string `json:"name"`                              // 应用名称（模糊搜索）
	Type     string `json:"type"`                              // 应用类型
	Status   string `json:"status"`                            // 状态
}
