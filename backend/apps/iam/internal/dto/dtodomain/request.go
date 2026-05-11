package dtodomain

import "github.com/morehao/golib/biz/gobject"

type CreateDomainReq struct {
	Domain string `json:"domain" binding:"required"`
}

type DomainPageListReq struct {
	gobject.PageQuery
	Domain string `json:"domain"`
}

type DeleteDomainReq struct {
	ID uint `json:"id" binding:"required"`
}
