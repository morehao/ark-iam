package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objtenant"
)

// ---------- 部门节点 ----------

type DepartmentCreateReq struct {
	ParentID string `json:"parentID" form:"parentID"` // 父节点ID,空为根节点
	objtenant.DepartmentBaseInfo
}

type DepartmentUpdateReq struct {
	DepartmentID string `json:"-" uri:"departmentID" binding:"required"` // 部门ID
	ParentID     string `json:"parentID" form:"parentID"`                // 父节点ID,空为根节点(改此字段=移动节点)
	objtenant.DepartmentBaseInfo
}

type DepartmentStatusReq struct {
	DepartmentID string               `json:"-" uri:"departmentID" binding:"required"` // 部门ID
	Status       model.DeptNodeStatus `json:"status" binding:"required"`               // 状态: enable-启用 disable-停用
}

type DepartmentDeleteReq struct {
	DepartmentID string `json:"-" uri:"departmentID" binding:"required"` // 部门ID
	Cascade      bool   `json:"cascade" form:"cascade"`                  // 是否级联删除子树与成员(默认拒绝)
}

type DepartmentTreeReq struct {
	Name   string               `json:"name" form:"name"`     // 部门名称过滤
	Status model.DeptNodeStatus `json:"status" form:"status"` // 状态过滤
}

// DepartmentChildrenReq 某部门直属子部门分页查询。
type DepartmentChildrenReq struct {
	DepartmentID string               `json:"-" uri:"departmentID" binding:"required"` // 部门ID(父节点)
	Name         string               `json:"name" form:"name"`                        // 部门名称过滤
	Status       model.DeptNodeStatus `json:"status" form:"status"`                    // 状态过滤
	Page         int                  `json:"page" form:"page"`                        // 页码
	PageSize     int                  `json:"pageSize" form:"pageSize"`                // 每页数量
}

// ---------- 部门关系 ----------

type DepartmentUserCreateReq struct {
	DepartmentID string                     `json:"-" uri:"departmentID" binding:"required"` // 部门ID
	UserID       string                     `json:"userID" binding:"required"`               // 用户ID
	RelationType model.DeptUserRelationType `json:"relationType"`                            // 关系类型: primary-行政主部门(每用户至多1) secondary-跨部门参与 leader-负责
}

type DepartmentUserUpdateReq struct {
	DepartmentID string                     `json:"-" uri:"departmentID" binding:"required"` // 部门ID
	UserID       string                     `json:"-" uri:"userID" binding:"required"`       // 用户ID
	RelationType model.DeptUserRelationType `json:"relationType"`                            // 关系类型
}

type DepartmentUserDeleteReq struct {
	DepartmentID string `json:"-" uri:"departmentID" binding:"required"` // 部门ID
	UserID       string `json:"-" uri:"userID" binding:"required"`       // 用户ID
}

type DepartmentUserPageListReq struct {
	Page         int                        `json:"page" form:"page"`                        // 页码
	PageSize     int                        `json:"pageSize" form:"pageSize"`                // 每页数量
	DepartmentID string                     `json:"-" uri:"departmentID" binding:"required"` // 部门ID
	RelationType model.DeptUserRelationType `json:"relationType" form:"relationType"`        // 关系类型过滤
	Keyword      string                     `json:"keyword" form:"keyword"`                  // 关键词(姓名/用户名/邮箱/手机 模糊)
}
