package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objtenant"
)

// ---------- 部门节点 ----------

type DepartmentCreateResp struct {
	DepartmentID string `json:"departmentID"` // 部门ID
}

type DepartmentTreeResp struct {
	List []DepartmentTreeItem `json:"list"` // 部门树
}

type DepartmentTreeItem struct {
	DepartmentID string `json:"departmentID"` // 部门ID
	ParentID     string `json:"parentID"`     // 父节点ID
	DeptPath     string `json:"deptPath"`     // 祖先链路径(含自身)
	DeptDepth    int    `json:"deptDepth"`    // 节点深度(根=1)
	CreatedAt    int64  `json:"createdAt"`    // 创建时间(unix 秒)
	objtenant.DepartmentBaseInfo
	Children []DepartmentTreeItem `json:"children"` // 子节点
}

// DepartmentChildrenResp 某部门直属子部门分页结果。
type DepartmentChildrenResp struct {
	List  []DepartmentChildItem `json:"list"`  // 直属子部门
	Total int64                 `json:"total"` // 总数
}

// DepartmentChildItem 子部门条目（不含 children，扁平展示）。
type DepartmentChildItem struct {
	DepartmentID string `json:"departmentID"` // 部门ID
	ParentID     string `json:"parentID"`     // 父节点ID
	DeptDepth    int    `json:"deptDepth"`    // 节点深度
	CreatedAt    int64  `json:"createdAt"`    // 创建时间(unix 秒)
	UpdatedAt    int64  `json:"updatedAt"`    // 更新时间(unix 秒)
	HasChildren  bool   `json:"hasChildren"`  // 是否还有下级
	objtenant.DepartmentBaseInfo
}

// ---------- 部门关系 ----------

type DepartmentUserCreateResp struct {
}

type DepartmentUserPageListResp struct {
	List  []DepartmentUserPageListItem `json:"list"`  // 关系列表
	Total int64                        `json:"total"` // 总数
}

type DepartmentUserPageListItem struct {
	DepartmentID string                     `json:"departmentID"` // 部门ID
	UserID       string                     `json:"userID"`       // 用户ID
	UserType     model.UserType             `json:"userType"`     // 账号类型(member真实用户/machine服务账号)
	UserName     string                     `json:"userName"`     // 用户姓名(租户内)
	Username     string                     `json:"username"`     // 全局用户名
	PrimaryEmail string                     `json:"primaryEmail"` // 主要邮箱
	PrimaryPhone string                     `json:"primaryPhone"` // 主要手机号
	Avatar       string                     `json:"avatar"`       // 头像URL
	IsSuspended  bool                       `json:"isSuspended"`  // 是否挂起
	RelationType model.DeptUserRelationType `json:"relationType"` // 关系类型
	JoinedAt     int64                      `json:"joinedAt"`     // 加入时间(关系创建时间)
}

// UserDepartmentItem 用户-部门归属条目（用户侧部门关系展示，含主/参与/负责）。
type UserDepartmentItem struct {
	DepartmentID   string                     `json:"departmentID"`   // 部门ID
	DepartmentName string                     `json:"departmentName"` // 部门名称
	RelationType   model.DeptUserRelationType `json:"relationType"`   // 关系类型
}
