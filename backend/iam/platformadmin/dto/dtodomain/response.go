package dtodomain

import "github.com/morehao/ark-iam/pkg/model"

type DomainCreateResp struct {
	ID string `json:"id"` // 域名ID
}

type DomainDetailResp struct {
	ID                 string                         `json:"id"`                 // 域名ID
	Domain             string                         `json:"domain"`             // 域名
	VerificationStatus model.DomainVerificationStatus `json:"verificationStatus"` // 验证状态: unverified-未验证, verified-已验证
	CreatedAt          int64                          `json:"createdAt"`          // 创建时间(unix 秒)
	UpdatedAt          int64                          `json:"updatedAt"`          // 更新时间(unix 秒)
}

type DomainPageListItem struct {
	ID                 string                         `json:"id"`                 // 域名ID
	Domain             string                         `json:"domain"`             // 域名
	VerificationStatus model.DomainVerificationStatus `json:"verificationStatus"` // 验证状态: unverified-未验证, verified-已验证
	CreatedAt          int64                          `json:"createdAt"`          // 创建时间(unix 秒)
	UpdatedAt          int64                          `json:"updatedAt"`          // 更新时间(unix 秒)
}

type DomainPageListResp struct {
	List  []DomainPageListItem `json:"list"`
	Total int64                `json:"total"`
}
