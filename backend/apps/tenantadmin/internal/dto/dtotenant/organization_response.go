package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
)

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
	CreatedAt      int64  `json:"createdAt"`      // 创建时间(unix 秒)
	objtenant.OrganizationBaseInfo
	Children []OrganizationTreeItem `json:"children"` // 子节点
}

// OrganizationChildrenResp 某部门直属子部门分页结果。
type OrganizationChildrenResp struct {
	List  []OrganizationChildItem `json:"list"`  // 直属子部门
	Total int64                   `json:"total"` // 总数
}

// OrganizationChildItem 子部门条目（不含 children，扁平展示）。
type OrganizationChildItem struct {
	OrganizationID string `json:"organizationID"` // 组织ID
	ParentID       string `json:"parentID"`       // 父节点ID
	OrgDepth       int    `json:"orgDepth"`       // 节点深度
	CreatedAt      int64  `json:"createdAt"`      // 创建时间(unix 秒)
	HasChildren    bool   `json:"hasChildren"`    // 是否还有下级
	objtenant.OrganizationBaseInfo
}

// ---------- 组织关系 ----------

type OrganizationUserCreateResp struct {
}

type OrganizationUserPageListResp struct {
	List  []OrganizationUserPageListItem `json:"list"`  // 关系列表
	Total int64                          `json:"total"` // 总数
}

type OrganizationUserPageListItem struct {
	OrganizationID string                    `json:"organizationID"` // 组织ID
	UserID         string                    `json:"userID"`         // 用户ID
	UserType       model.UserType            `json:"userType"`       // 账号类型(member真实用户/machine服务账号)
	UserName       string                    `json:"userName"`       // 用户姓名(租户内)
	Username       string                    `json:"username"`       // 全局用户名
	PrimaryEmail   string                    `json:"primaryEmail"`   // 主要邮箱
	PrimaryPhone   string                    `json:"primaryPhone"`   // 主要手机号
	Avatar         string                    `json:"avatar"`         // 头像URL
	IsSuspended    bool                      `json:"isSuspended"`    // 是否挂起
	RelationType   model.OrgUserRelationType `json:"relationType"`   // 关系类型
	JoinedAt       int64                     `json:"joinedAt"`       // 加入时间(关系创建时间)
}

// UserOrganizationItem 用户-组织归属条目（用户侧组织关系展示，含主/参与/负责）。
type UserOrganizationItem struct {
	OrganizationID   string                    `json:"organizationID"`   // 组织ID
	OrganizationName string                    `json:"organizationName"` // 组织名称
	RelationType     model.OrgUserRelationType `json:"relationType"`     // 关系类型
}
