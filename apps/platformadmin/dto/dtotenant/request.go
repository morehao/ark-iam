package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/ark-iam/pkg/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateReq struct {
	objtenant.TenantBaseInfo
	Admin *TenantAdminCreateReq `json:"admin" binding:"required"` // 租户管理员(必填：每个租户都必须有管理员)
	// PrimaryDomain 仅 dedicated 新建时使用：同事务创建该 primary（draft 域名）。
	// 普通(shared)建租户忽略。见 p0-tenant-lifecycle-contract.md §2。
	PrimaryDomain string `json:"primaryDomain"`
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// TenantAdminCreateReq 建租户时一并创建的租户管理员。
// 不含密码：初始临时密码由系统生成并仅在创建响应中返回一次
// （见 docs/design/system-design.md §5.8/D6）。
type TenantAdminCreateReq struct {
	Name         string `json:"name" binding:"required"` // 姓名(必填)
	Username     string `json:"username"`                // 全局用户名(可选)
	PrimaryEmail string `json:"primaryEmail"`            // 主要邮箱(与手机号至少一个)
	PrimaryPhone string `json:"primaryPhone"`            // 主要手机号(与邮箱至少一个)
}

// TenantAdminResetPasswordReq 重置租户内置管理员(builtin)密码；tenantID 只来自 path。
type TenantAdminResetPasswordReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type TenantUpdateReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
	objtenant.TenantBaseInfo
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

type TenantDetailReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
}

type TenantPageListReq struct {
	gobject.PageQuery
	Name string `json:"name" form:"name"` // 租户名称（模糊搜索）
	// Status 状态筛选（active-正常 / suspended-已挂起）；空值表示不筛选。
	Status model.TenantStatus `json:"status" form:"status"`
}

type TenantDeleteReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// TenantOffboardReq offboard 租户（P0 唯一停用入口）：tenant→suspended、primary disabled、
// 清 issuer、撤 session/refresh；接受 provisioning/active。D DELETE 一律 403，只能走此入口。
type TenantOffboardReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"`
	Operator objidentity.Operator
}
