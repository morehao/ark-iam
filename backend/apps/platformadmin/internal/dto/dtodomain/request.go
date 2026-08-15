package dtodomain

import "github.com/morehao/golib/biz/gobject"

type DomainCreateReq struct {
	Domain string `json:"domain" binding:"required"` // 域名
}

type DomainUpdateReq struct {
	DomainID   string `json:"-" uri:"domainID" binding:"required"` // 域名ID
	Domain     string `json:"domain"`                              // 域名
	IsVerified *bool  `json:"isVerified"`                          // 是否验证(0-未验证 1-已验证)
}

type DomainDetailReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"` // 域名ID
}

type DomainPageListReq struct {
	gobject.PageQuery
	Domain string `json:"domain" form:"domain"` // 域名(模糊搜索)
}

type DomainDeleteReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"` // 域名ID
}
