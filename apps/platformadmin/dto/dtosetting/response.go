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
