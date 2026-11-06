package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
)

type UserPageListReq struct {
	Page         int              `json:"page" form:"page"`                 // 页码
	PageSize     int              `json:"pageSize" form:"pageSize"`         // 每页数量
	Keyword      string           `json:"keyword" form:"keyword"`           // 关键词(姓名/用户名/邮箱/手机 模糊)
	Status       model.UserStatus `json:"status" form:"status"`             // 状态过滤(active正常/suspended挂起;空=不过滤)
	DepartmentID string           `json:"departmentID" form:"departmentID"` // 部门ID(仅筛选恰在该部门的用户,不含子部门)
}

// UserCreateReq 创建租户成员。
// 不接收密码：新建自然人的初始临时密码由服务端生成、仅创建响应返回一次，且该成员首次登录必须改密
// （见 docs/design/system-design.md §5.8/D6/D7）。
type UserCreateReq struct {
	PersonID               string           `json:"personID"`                               // 已有自然人ID(可选,优先关联)
	Username               string           `json:"username"`                               // 全局用户名(可选)
	PrimaryEmail           string           `json:"primaryEmail"`                           // 主要邮箱
	PrimaryPhone           string           `json:"primaryPhone"`                           // 主要手机号
	Name                   string           `json:"name" binding:"required"`                // 姓名(新建 person 时的自然人姓名)
	Avatar                 string           `json:"avatar"`                                 // 头像URL
	Status                 model.UserStatus `json:"status"`                                 // 状态(active正常/suspended挂起;空=active)
	PrimaryDepartmentID    string           `json:"primaryDepartmentID" binding:"required"` // 行政主部门ID(primary,单值,必传:用户必须从属部门)
	SecondaryDepartmentIDs []string         `json:"secondaryDepartmentIDs"`                 // 参与部门ID列表(secondary,可多条,可选)
	LeaderDepartmentIDs    []string         `json:"leaderDepartmentIDs"`                    // 负责部门ID列表(leader,可多条,可选;每部门至多1负责人)
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type UserDetailReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserUpdateReq struct {
	UserID                 string            `json:"-" uri:"userID" binding:"required"` // 用户ID
	Username               *string           `json:"username"`                          // 用户名(nil=不变)
	PrimaryEmail           *string           `json:"primaryEmail"`                      // 主要邮箱(nil=不变)
	PrimaryPhone           *string           `json:"primaryPhone"`                      // 主要手机号(nil=不变)
	Name                   string            `json:"name"`                              // 姓名
	Avatar                 string            `json:"avatar"`                            // 头像URL
	Status                 *model.UserStatus `json:"status"`                            // 状态(active正常/suspended挂起;nil=不变)
	PrimaryDepartmentID    *string           `json:"primaryDepartmentID"`               // 主部门(primary,nil=不变;非nil=替换主部门,不可清空)
	SecondaryDepartmentIDs *[]string         `json:"secondaryDepartmentIDs"`            // 参与部门(secondary,nil=不变;[]=清空;含值=全量替换)
	LeaderDepartmentIDs    *[]string         `json:"leaderDepartmentIDs"`               // 负责部门(leader,nil=不变;[]=清空;含值=全量替换;每部门至多1负责人)
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// UserResetPasswordReq 重置成员密码：不接收密码，由服务端生成临时密码并在响应中返回一次。
type UserResetPasswordReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type UserRolesListReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID
}

type UserRolesUpdateReq struct {
	UserID  string   `json:"-" uri:"userID" binding:"required"` // 用户ID
	AppID   string   `json:"appID"`                             // 目标应用ID(按应用授权；空串=系统/未归属应用组)
	RoleIDs []string `json:"roleIDs" binding:"required"`        // 角色ID列表(全量替换该应用下的授权)
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ---------- 应用准入（用户子资源） ----------
// 准入回答"该成员能否使用某应用"，与角色（进入后能做什么）解耦：无角色应用同样在此授权。

type UserApplicationsListReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID（path）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// UserApplicationsUpdateReq 全量替换该成员的准入应用集合（PUT 集合语义）：
// 集合内 enable、集合外 disable，并对集合外应用级联撤销其角色绑定（不变式 I2）。
// AppIDs 用指针 + required：字段必须显式传入（杜绝漏传被当成"清空全部准入"），
// 而传 `[]` 是合法的"清空全部准入"。
type UserApplicationsUpdateReq struct {
	UserID string    `json:"-" uri:"userID" binding:"required"` // 用户ID（path）
	AppIDs *[]string `json:"appIDs" binding:"required"`         // 准入应用ID全集（[]=清空全部准入）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ApplicationUsersListReq 应用视角：被准入的成员列表（谁能进这个应用）。
type ApplicationUsersListReq struct {
	AppID string `json:"-" uri:"appID" binding:"required"` // 应用ID（path）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ---------- 第三方身份（用户子资源） ----------
// 租户维度一律取自登录上下文，不接受请求体传入 tenantID，避免参数污染。

type UserIdentityListReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID（path）
}

type UserIdentityCreateReq struct {
	UserID     string                   `json:"-" uri:"userID" binding:"required"` // 用户ID（path）
	Issuer     string                   `json:"issuer" binding:"required"`         // 身份提供商
	IdentityID string                   `json:"identityID" binding:"required"`     // 第三方用户ID
	Detail     model.UserIdentityDetail `json:"detail"`                            // 详细信息（写入白名单以 model 载具类型为准）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type UserIdentityDeleteReq struct {
	UserID         string `json:"-" uri:"userID" binding:"required"`     // 用户ID（path）
	UserIdentityID string `json:"-" uri:"identityID" binding:"required"` // 用户身份ID（path）
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// ---------- 登录日志（用户子资源） ----------

type UserLoginLogListReq struct {
	UserID string `json:"-" uri:"userID" binding:"required"` // 用户ID（path）
}
