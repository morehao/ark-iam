package dtosetting

// SettingItem 配置项对外契约（敏感项 value 为掩码）。
type SettingItem struct {
	ID             string `json:"id"`             // 配置项ID
	Group          string `json:"group"`          // 配置分组
	Key            string `json:"key"`            // 键名
	ValueType      string `json:"valueType"`      // 值类型
	Value          string `json:"value"`          // 值（敏感项为掩码）
	EncryptionMode string `json:"encryptionMode"` // plain/encrypted
	Description    string `json:"description"`    // 描述
	Status         string `json:"status"`         // enabled/disabled
	CreatedAt      int64  `json:"createdAt"`      // 创建时间（unix 秒）
	UpdatedAt      int64  `json:"updatedAt"`      // 更新时间（unix 秒）
}

// SettingDetailResp 配置项详情响应。
type SettingDetailResp struct {
	SettingItem
}

// SettingPageListResp 配置项分页列表响应。
type SettingPageListResp struct {
	List  []*SettingItem `json:"list"`
	Total int64          `json:"total"`
}

// SettingGroupMeta 是配置分组元数据（供前端渲染分组下拉与说明）。
// 单一事实源在 svcsetting.settingGroups：新增一类运行时配置只改那一处，前端零改动。
//
// 分组以 Description（中文说明）对外展示，不额外维护硬编码的 Title——避免中英/标题与说明两套文案漂移。
type SettingGroupMeta struct {
	Group       string `json:"group"`       // 分组标识，如 notify
	Description string `json:"description"` // 分组说明（下拉/筛选项展示用）
}

// SettingGroupListResp 分组元数据列表响应。
type SettingGroupListResp struct {
	List []*SettingGroupMeta `json:"list"`
}
