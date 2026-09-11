package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type TenantCreateReq struct {
	objtenant.TenantBaseInfo
	Admin *TenantAdminCreateReq `json:"admin" binding:"required"` // 租户管理员(必填：每个租户都必须有管理员)
}

// TenantAdminCreateReq 建租户时一并创建的租户管理员。
// 不含密码：初始临时密码由系统生成并仅在创建响应中返回一次
// （见 docs/design/tenant-admin-provisioning-design-20260912.md D3/D6）。
type TenantAdminCreateReq struct {
	Name         string `json:"name" binding:"required"` // 姓名(必填)
	Username     string `json:"username"`                // 全局用户名(可选)
	PrimaryEmail string `json:"primaryEmail"`            // 主要邮箱(与手机号至少一个)
	PrimaryPhone string `json:"primaryPhone"`            // 主要手机号(与邮箱至少一个)
}

// TenantAdminResetPasswordReq 重置租户内置管理员(builtin)密码；tenantID 只来自 path。
type TenantAdminResetPasswordReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
}

type TenantUpdateReq struct {
	TenantID string `json:"-" uri:"tenantID" binding:"required"` // 租户ID
	objtenant.TenantBaseInfo
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
}

// ---------- 日志（审计记录） ----------

type LogDetailReq struct {
	LogID string `json:"-" uri:"logID" binding:"required"` // 日志ID
}

type LogPageListReq struct {
	gobject.PageQuery
	TenantID string `json:"tenantID" form:"tenantID"` // 租户ID
	Key      string `json:"key" form:"key"`           // 日志键
}
