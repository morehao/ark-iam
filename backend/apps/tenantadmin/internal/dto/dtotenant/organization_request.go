package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
)

// ---------- 组织节点 ----------

type OrganizationCreateReq struct {
	ParentID string `json:"parentID" form:"parentID"` // 父节点ID,空为根节点
	objtenant.OrganizationBaseInfo
}

type OrganizationUpdateReq struct {
	OrganizationID string `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	ParentID       string `json:"parentID" form:"parentID"`                  // 父节点ID,空为根节点(改此字段=移动节点)
	objtenant.OrganizationBaseInfo
}

type OrganizationStatusReq struct {
	OrganizationID string `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	Status         string `json:"status" binding:"required"`                 // 状态: active-启用 inactive-停用
}

type OrganizationDeleteReq struct {
	OrganizationID string `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	Cascade        bool   `json:"cascade" form:"cascade"`                    // 是否级联删除子树与成员(默认拒绝)
}

type OrganizationTreeReq struct {
	Name   string `json:"name" form:"name"`     // 组织名称过滤
	Status string `json:"status" form:"status"` // 状态过滤
}

// OrganizationChildrenReq 某部门直属子部门分页查询。
type OrganizationChildrenReq struct {
	OrganizationID string `json:"-" uri:"organizationID" binding:"required"` // 部门ID(父节点)
	Name           string `json:"name" form:"name"`                          // 部门名称过滤
	Status         string `json:"status" form:"status"`                      // 状态过滤
	Page           int    `json:"page" form:"page"`                          // 页码
	PageSize       int    `json:"pageSize" form:"pageSize"`                  // 每页数量
}

// ---------- 组织关系 ----------

type OrganizationUserCreateReq struct {
	OrganizationID string                    `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	UserID         string                    `json:"userID" binding:"required"`                 // 用户ID
	RelationType   model.OrgUserRelationType `json:"relationType"`                              // 关系类型: primary-行政主部门(每用户至多1) secondary-跨部门参与 leader-负责
}

type OrganizationUserUpdateReq struct {
	OrganizationID string                    `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	UserID         string                    `json:"-" uri:"userID" binding:"required"`         // 用户ID
	RelationType   model.OrgUserRelationType `json:"relationType"`                              // 关系类型
}

type OrganizationUserDeleteReq struct {
	OrganizationID string `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	UserID         string `json:"-" uri:"userID" binding:"required"`         // 用户ID
}

type OrganizationUserPageListReq struct {
	Page           int                       `json:"page" form:"page"`                          // 页码
	PageSize       int                       `json:"pageSize" form:"pageSize"`                  // 每页数量
	OrganizationID string                    `json:"-" uri:"organizationID" binding:"required"` // 组织ID
	RelationType   model.OrgUserRelationType `json:"relationType" form:"relationType"`          // 关系类型过滤
	Keyword        string                    `json:"keyword" form:"keyword"`                    // 关键词(姓名/用户名/邮箱/手机 模糊)
}
