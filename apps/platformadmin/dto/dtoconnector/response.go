package dtoconnector

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/golib/biz/gobject"
)

// ConnectorCreateResp 创建连接器响应。
type ConnectorCreateResp struct {
	ConnectorID string `json:"connectorID"` // 连接器ID
}

// ConnectorDetailResp 连接器详情响应。
// 内嵌 objauth.ConnectorBaseInfo（含 defaultTenantID/name/displayName/protocol/provider/status/
// 四个开关/config/claimMapping/domainPolicy），并补 defaultTenantName 与 createdAt/updatedAt。
type ConnectorDetailResp struct {
	ConnectorID string `json:"connectorID"` // 连接器ID
	objauth.ConnectorBaseInfo
	DefaultTenantName string `json:"defaultTenantName"` // 默认落租户名称（回显用，缺失为空串）
	gobject.OperatorBaseInfo
}

// ConnectorPageListItem 连接器列表项（字段同详情）。
type ConnectorPageListItem struct {
	ConnectorID string `json:"connectorID"` // 连接器ID
	objauth.ConnectorBaseInfo
	DefaultTenantName string `json:"defaultTenantName"` // 默认落租户名称（回显用，缺失为空串）
	gobject.OperatorBaseInfo
}

// ConnectorPageListResp 连接器列表响应。
type ConnectorPageListResp struct {
	List  []ConnectorPageListItem `json:"list"`  // 列表数据
	Total int64                   `json:"total"` // 总数
}

// ConnectorFactoryResp 连接器工厂定义响应（前端按 configSchema 渲染表单）。
type ConnectorFactoryResp struct {
	FactoryID     string                     `json:"factoryID"`     // 工厂标识
	Protocol      model.ConnectorProtocol    `json:"protocol"`      // 协议类型
	Provider      model.ConnectorProvider    `json:"provider"`      // 提供商
	DisplayName   string                     `json:"displayName"`   // 显示名称
	IsStandard    bool                       `json:"isStandard"`    // 是否标准连接器
	DefaultScopes []string                   `json:"defaultScopes"` // 默认 scope
	Capabilities  []model.ConnectorCapability `json:"capabilities"` // 能力标识
	ConfigSchema  map[string]any             `json:"configSchema"`  // 配置 schema（前端按 ConnectorConfigSchema {required?:string[]} 解析）
}

// ConnectorFactoryListResp 连接器工厂列表响应。
type ConnectorFactoryListResp struct {
	List []ConnectorFactoryResp `json:"list"` // 工厂列表
}

// ConnectorTestResp 连接器连通性测试响应。
type ConnectorTestResp struct {
	Success bool   `json:"success"` // 是否连通
	Message string `json:"message"` // 结果说明
}
