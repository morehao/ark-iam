package dtodomain

import "github.com/morehao/ark-iam/pkg/model"

type DomainCreateResp struct {
	ID string `json:"id"`
}

type DomainActionResp struct {
	DomainID string             `json:"domainId"`
	Status   model.DomainStatus `json:"status"`
}

type DomainDetailResp struct {
	ID                 string                         `json:"id"`
	TenantID           string                         `json:"tenantId"`
	Domain             string                         `json:"domain"`
	DomainRole         model.DomainRole               `json:"domainRole"`
	DomainUsage        model.DomainUsage              `json:"domainUsage"`
	TargetDomainID     string                         `json:"targetDomainId"`
	Status             model.DomainStatus             `json:"status"`
	VerificationMethod model.DomainVerificationMethod `json:"verificationMethod"`
	VerifiedAt         *int64                         `json:"verifiedAt"` // unix 秒，空为 null
	TLSReadyAt         *int64                         `json:"tlsReadyAt"`
	RetiredAt          *int64                         `json:"retiredAt"`
	ReleasedAt         *int64                         `json:"releasedAt"`
	RetireReason       string                         `json:"retireReason"`
	ReleaseApproval    string                         `json:"releaseApproval"`
	CreatedAt          int64                          `json:"createdAt"`
	UpdatedAt          int64                          `json:"updatedAt"`
}

type DomainPageListItem struct {
	ID          string             `json:"id"`
	TenantID    string             `json:"tenantId"`
	Domain      string             `json:"domain"`
	DomainRole  model.DomainRole   `json:"domainRole"`
	DomainUsage model.DomainUsage  `json:"domainUsage"`
	Status      model.DomainStatus `json:"status"`
	CreatedAt   int64              `json:"createdAt"`
	UpdatedAt   int64              `json:"updatedAt"`
}

type DomainPageListResp struct {
	List  []DomainPageListItem `json:"list"`
	Total int64                `json:"total"`
}
