package dtodomain

import "github.com/morehao/golib/biz/gobject"

type CreateDomainReq struct {
	Domain string `json:"domain" binding:"required"` // 域名
}

type UpdateDomainReq struct {
	ID         uint   `json:"id" binding:"required"` // 域名ID
	Domain     string `json:"domain"`                // 域名
	IsVerified *int8  `json:"isVerified"`            // 是否验证(0-未验证 1-已验证)
}

type DomainDetailReq struct {
	ID uint `form:"id" binding:"required"` // 域名ID
}

type DomainPageListReq struct {
	gobject.PageQuery
	Domain string `json:"domain"` // 域名(模糊搜索)
}

type DeleteDomainReq struct {
	ID uint `json:"id" binding:"required"` // 域名ID
}
