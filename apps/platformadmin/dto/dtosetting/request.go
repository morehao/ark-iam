package dtosetting

import (
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gobject"
)

// SettingCreateReq 创建配置项（同 (group,key) upsert）。
type SettingCreateReq struct {
	Group       string `json:"group" binding:"required"` // 配置分组（白名单内）
	Key         string `json:"key" binding:"required"`   // 键名（同组内唯一）
	ValueType   string `json:"valueType"`                // json/toml/yaml/string/int/bool/float，默认 string
	Value       string `json:"value" binding:"required"` // 值（明文；敏感项置 encrypted=true 加密落库）
	Encrypted   bool   `json:"encrypted"`                // 是否加密落库
	Status      string `json:"status"`                   // enabled/disabled，默认 enabled
	Description string `json:"description"`              // 描述
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// SettingUpdateReq 局部更新配置项（零值字段不更新）。
//
// Value 传掩码值（MaskedValue）表示「不修改该字段」——敏感项回显掩码，回存时保持原值。
type SettingUpdateReq struct {
	SettingID   string `json:"-" uri:"settingID" binding:"required"` // 配置项ID
	Value       string `json:"value"`                                // 新值；等于掩码表示不改
	Encrypted   bool   `json:"encrypted"`                            // 配合 value：是否加密落库
	Status      string `json:"status"`                               // enabled/disabled
	Description string `json:"description"`                          // 描述
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// SettingUpdateStatusReq 启停配置项。
type SettingUpdateStatusReq struct {
	SettingID string `json:"-" uri:"settingID" binding:"required"` // 配置项ID
	Status    string `json:"status" binding:"required"`            // enabled/disabled
	// Operator 由 controller 从鉴权结果填入（json:"-"，客户端不可伪造）。
	Operator objidentity.Operator
}

// SettingDetailReq 配置项详情。
type SettingDetailReq struct {
	SettingID string `json:"-" uri:"settingID" binding:"required"` // 配置项ID
}

// SettingPageListReq 配置项分页列表（均可选）。
type SettingPageListReq struct {
	gobject.PageQuery
	Group     string `json:"group" form:"group"`         // 分组（白名单内，缺省不限白名单内全部分组）
	Key       string `json:"key" form:"key"`             // 键名（模糊）
	ValueType string `json:"valueType" form:"valueType"` // 值类型
	Status    string `json:"status" form:"status"`       // 状态
}

// SettingDeleteReq 删除配置项。
type SettingDeleteReq struct {
	SettingID string `json:"-" uri:"settingID" binding:"required"` // 配置项ID
	// Operator 由 controller 从鉴权结果填入（json:"-"）。
	Operator objidentity.Operator
}
