package dtodomain

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gobject"
)

type DomainCreateReq struct {
	Domain string `json:"domain" binding:"required"` // 域名
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type DomainUpdateReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"` // 域名ID
	Domain   string `json:"domain"`                              // 域名
	// VerificationStatus 验证状态: unverified-未验证, verified-已验证；留空表示不修改。
	VerificationStatus model.DomainVerificationStatus `json:"verificationStatus"`
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
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
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}
