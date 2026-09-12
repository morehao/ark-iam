package objtenant

import "github.com/morehao/ark-iam/pkg/model"

type TenantBaseInfo struct {
	// Code 租户编码：由服务端自动生成（pkg/core/tenant.GenerateCode，t_<12 位随机 hex>，
	// 例 t_3f7a9c1d2e4b），创建/更新入参传入无效，创建后不可修改；仅在明细/列表出参中回显。
	Code   string `json:"code" form:"code"`     // 租户编码(服务端生成,入参忽略)
	DbUser string `json:"dbUser" form:"dbUser"` // 数据库用户
	Name   string `json:"name" form:"name"`     // 租户名称
	// Status 租户状态：active-正常 / suspended-已挂起（见 model.TenantStatus）。
	// 仅 active 允许该租户成员登录与签发令牌。
	Status model.TenantStatus `json:"status" form:"status"` // 租户状态
	Tag    string             `json:"tag" form:"tag"`       // 标签
	Type   model.TenantType   `json:"type" form:"type"`     // 租户类型: customer-客户租户, platform-平台租户(分类标识,不参与隔离判定)
}
