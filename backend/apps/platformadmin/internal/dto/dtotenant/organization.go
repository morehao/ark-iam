package dtotenant

import "github.com/morehao/ark-iam/pkg/iam/object/objtenant"

// ---------- 组织只读（平台视角） ----------

type OrganizationTreeReq struct {
	TenantID string `json:"tenantID" form:"tenantID" binding:"required"` // 租户ID（必填，跨租户排查）
}

type OrganizationTreeResp struct {
	List []OrganizationTreeItem `json:"list"` // 组织树
}

type OrganizationTreeItem struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	ParentID       string `json:"parentID"`       // 父节点ID
	OrgPath        string `json:"orgPath"`        // 祖先链路径(含自身)
	OrgDepth       int    `json:"orgDepth"`       // 节点深度(根=1)
	objtenant.OrganizationBaseInfo
	Children []OrganizationTreeItem `json:"children"` // 子节点
}

// ---------- 用户组织归属只读 ----------

type UserOrganizationListReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserOrganizationListResp struct {
	List []UserOrganizationItem `json:"list"` // 用户组织归属
}

type UserOrganizationItem struct {
	OrganizationID   string `json:"organizationID"`   // 组织ID
	OrganizationName string `json:"organizationName"` // 组织名称
	RelationType     string `json:"relationType"`     // 关系类型
	IsPrimary        bool   `json:"isPrimary"`        // 是否主归属
}
