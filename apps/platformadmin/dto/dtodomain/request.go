package dtodomain

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gobject"
)

// DomainCreateReq 创建域名（平台端）。P0 只创建 draft；状态流转走动作端点。
// D alias 必填 targetDomain（host，服务端解析为同租户 target_domain_id）。
// primary 只允许 usage=login/both。
type DomainCreateReq struct {
	TenantID           string                         `json:"tenantId" binding:"required"` // 目标租户ID
	Domain             string                         `json:"domain" binding:"required"`   // 归一化 host
	DomainRole         model.DomainRole               `json:"domainRole" binding:"required"`
	DomainUsage        model.DomainUsage              `json:"domainUsage" binding:"required"`
	TargetDomain       string                         `json:"targetDomain"`       // alias/email 目标 primary 的 host（服务端解析）
	VerificationMethod model.DomainVerificationMethod `json:"verificationMethod"` // manual/dns_txt（默认 manual）
	Operator           objidentity.Operator
}

type DomainDetailReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"`
}

type DomainPageListReq struct {
	gobject.PageQuery
	TenantID    string             `json:"tenantId" form:"tenantId"`
	Domain      string             `json:"domain" form:"domain"`
	Status      model.DomainStatus `json:"status" form:"status"`
	DomainRole  model.DomainRole   `json:"domainRole" form:"domainRole"`
	DomainUsage model.DomainUsage  `json:"domainUsage" form:"domainUsage"`
}

// DomainDeleteReq 删除（仅 draft/verified/tls_ready；被未 released alias 引用时拒绝；active 必须走 disable/offboard）。
type DomainDeleteReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"`
	Operator objidentity.Operator
}

// 动作端点请求（路径参数 domainID 是唯一 ID 源）。全部含 Operator 供审计。

type DomainVerifyReq struct {
	DomainID string                         `json:"-" uri:"domainID" binding:"required"`
	Method   model.DomainVerificationMethod `json:"method"` // manual/dns_txt；空按 manual
	Token    string                         `json:"token"`  // DNS TXT 挑战值（可选）
	Operator objidentity.Operator
}

type DomainTLSReadyReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"`
	CertRef  string `json:"certRef"` // 证书引用（P0 运维确认，仅记录）
	Operator objidentity.Operator
}

type DomainActivateReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"`
	Operator objidentity.Operator
}

type DomainDisableReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"`
	Operator objidentity.Operator
}

type DomainEnableReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"`
	Operator objidentity.Operator
}

type DomainRetireReq struct {
	DomainID string `json:"-" uri:"domainID" binding:"required"`
	Reason   string `json:"reason"`
	Operator objidentity.Operator
}

type DomainReleaseReq struct {
	DomainID       string `json:"-" uri:"domainID" binding:"required"`
	ApprovalTicket string `json:"approvalTicket"` // 正式释放审批单；取消不发
	Operator       objidentity.Operator
}
