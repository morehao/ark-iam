package dtordirectory

// Member 是成员摘要（对外只读视图）。
//
// 字段只增不改（设计文档「接口契约」）：新增字段必须是可选字段，
// SDK 侧忽略未知字段，因此"SDK 版本旧"不会导致调用失败。
type Member struct {
	UserID          string   `json:"userID"`
	Name            string   `json:"name"`
	Avatar          string   `json:"avatar"`
	Status          string   `json:"status"`   // active / suspended
	UserType        string   `json:"userType"` // member / machine
	DepartmentNames []string `json:"departmentNames"`
}

// MemberListResp 是批量成员查询响应。
//
// Missing 中的 id 与"跨租户 id"不可区分（刻意防枚举）：
// 调用方不得据此判断某个 id 是否存在于其它租户。
type MemberListResp struct {
	List    []Member `json:"list"`
	Missing []string `json:"missing"`
}

// Department 是部门树节点。
type Department struct {
	ID       string       `json:"id"`
	ParentID string       `json:"parentID"`
	Name     string       `json:"name"`
	Children []Department `json:"children,omitempty"`
}

// Role 是角色清单条目，补齐 ID token `groups` 只有编码的缺口。
type Role struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	AppID string `json:"appID"`
}

// RoleListResp 是角色清单响应。单个响应无上界，因此显式声明是否被截断。
type RoleListResp struct {
	List      []Role `json:"list"`
	Truncated bool   `json:"truncated"`
}

// MaxBatchMemberIDs 是批量成员查询的 id 上限（超出显式 400，不静默截断）。
const MaxBatchMemberIDs = 100

// MaxRoleListSize 是角色清单的返回上限；超出置 Truncated=true。
const MaxRoleListSize = 500
