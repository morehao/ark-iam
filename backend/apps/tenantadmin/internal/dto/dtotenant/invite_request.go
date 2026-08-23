package dtotenant

import "github.com/morehao/golib/biz/gobject"

// InviteCreateReq 生成一条租户加入邀请。凭证持有者凭 inviteCode 自助加入该租户（非 owner）。
type InviteCreateReq struct {
	// ExpireHours 有效期小时数（0 表示不过期）
	ExpireHours int `json:"expireHours"`
}

type InviteCreateResp struct {
	InviteID string `json:"inviteID"` // 邀请ID
	Code     string `json:"code"`     // 邀请码（一次性，凭此加入）
}

type InviteRevokeReq struct {
	InviteID string `json:"-" uri:"inviteID" binding:"required"` // 邀请ID
}

type InvitePageListReq struct {
	gobject.PageQuery
	Status string `json:"status" form:"status"` // 状态过滤
}

type InvitePageListItem struct {
	InviteID  string `json:"inviteID"`  // 邀请ID
	Code      string `json:"code"`      // 邀请码
	Status    string `json:"status"`    // 状态
	ExpiresAt *int64 `json:"expiresAt"` // 过期时间（秒时间戳，可空）
	CreatedAt int64  `json:"createdAt"` // 创建时间（秒时间戳）
}

type InvitePageListResp struct {
	List  []InvitePageListItem `json:"list"`  // 数据列表
	Total int64                `json:"total"` // 数据总条数
}
