package dtoappdefinition

type CreateReq struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl"`
	HomepageURL string `json:"homepageUrl"`
	Type        string `json:"type"`
	Sort        int    `json:"sort"`
}

type UpdateReq struct {
	AppDefID    uint   `json:"appDefId" binding:"required"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl"`
	HomepageURL string `json:"homepageUrl"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Sort        int    `json:"sort"`
}

type DetailReq struct {
	AppDefID uint `form:"appDefId" binding:"required"`
}

type DeleteReq struct {
	AppDefID uint `json:"appDefId" binding:"required"`
}

type PageListReq struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Name     string `form:"name"`
	Type     string `form:"type"`
	Status   string `form:"status"`
}
