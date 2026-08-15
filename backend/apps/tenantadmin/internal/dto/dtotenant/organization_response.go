package dtotenant

import "github.com/morehao/ark-iam/pkg/iam/object/objtenant"

// ---------- 组织节点 ----------

type OrganizationCreateResp struct {
	OrganizationID string `json:"organizationID"` // 组织ID
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

type OrganizationDetailResp struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	ParentID       string `json:"parentID"`       // 父节点ID
	OrgPath        string `json:"orgPath"`        // 祖先链路径(含自身)
	OrgDepth       int    `json:"orgDepth"`       // 节点深度(根=1)
	objtenant.OrganizationBaseInfo
	Ancestors []OrganizationAncestor `json:"ancestors"` // 祖先链(面包屑,自顶向下)
}

type OrganizationAncestor struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	Name           string `json:"name"`           // 组织名称
}

// ---------- 组织关系 ----------

type OrganizationUserCreateResp struct {
}

type OrganizationUserPageListResp struct {
	List  []OrganizationUserPageListItem `json:"list"`  // 关系列表
	Total int64                          `json:"total"` // 总数
}

type OrganizationUserPageListItem struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	UserID         string `json:"userID"`         // 用户ID
	UserName       string `json:"userName"`       // 用户姓名(租户内)
	Username       string `json:"username"`       // 全局用户名
	PrimaryEmail   string `json:"primaryEmail"`   // 主要邮箱
	PrimaryPhone   string `json:"primaryPhone"`   // 主要手机号
	Avatar         string `json:"avatar"`         // 头像URL
	IsSuspended    bool   `json:"isSuspended"`    // 是否挂起
	RelationType   string `json:"relationType"`   // 关系类型
	IsPrimary      bool   `json:"isPrimary"`      // 是否主归属
	JoinedAt       int64  `json:"joinedAt"`       // 加入时间(关系创建时间)
}

type OrganizationSubtreeUsersResp struct {
	List []OrganizationSubtreeUser `json:"list"` // 子树成员(去重)
}

type OrganizationSubtreeUser struct {
	UserID   string `json:"userID"`   // 用户ID
	UserName string `json:"userName"` // 用户姓名
}

// ---------- 用户归属 ----------

type UserOrganizationListResp struct {
	List []UserOrganizationItem `json:"list"` // 用户组织归属
}

type UserOrganizationItem struct {
	OrganizationID   string `json:"organizationID"`   // 组织ID
	OrganizationName string `json:"organizationName"` // 组织名称
	RelationType     string `json:"relationType"`     // 关系类型
	IsPrimary        bool   `json:"isPrimary"`        // 是否主归属
}

type UserOrganizationsUpdateResp struct {
}
