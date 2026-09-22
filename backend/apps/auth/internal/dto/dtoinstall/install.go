package dtoinstall

// 初始化接口的 DTO。
//
// 命名遵循 `<业务名词><动词>Req/Resp`，ID 一律 string、时间一律秒级 int64。
// 注意 **Resp 里绝不出现口令**：管理员口令由运维自选，服务端既不知道明文也不必回显。

// InstallationStatusResp 是 GET /install/status 的响应：公开、只读、不含敏感信息。
//
// 三个字段各自回答一个具体的部署问题，缺一个都会让部署期只能靠猜：
//   - Initialized：系统是否已完成首次初始化（决定页面渲染表单还是"已完成"页）；
//   - TokenRequired：服务端是否配置了 BOOTSTRAP_TOKEN。**提前告知**，而不是让运维
//     填完三步表单、点了提交才收到 503；
//   - SchemaReady：表结构是否就绪。把"db.auto_migrate=false + 空库"这个部署失败模式
//     变成可执行提示（"先把表建出来"），而不是一个 500 或一个永远建不成的初始化。
//
// 刻意**不返回版本信息**：未认证端点暴露版本号是无收益的信息泄露。
type InstallationStatusResp struct {
	Initialized   bool                 `json:"initialized"`
	TokenRequired bool                 `json:"tokenRequired"`
	SchemaReady   bool                 `json:"schemaReady"`
	Consoles      InstallationConsoles `json:"consoles"`
}

// InstallationConsoles 是两个内置控制台的对外地址，供初始化页给出"稍后从哪登录"。
//
// 与 L1 写进回调白名单的地址同源（同一份配置），因此页面看到的入口必然可用。
type InstallationConsoles struct {
	PlatformAdminWeb string `json:"platformAdminWeb"` // 平台管理后台地址（登录页）
	TenantAdminWeb   string `json:"tenantAdminWeb"`   // 租户管理后台地址（登录页）
}

// InstallationInitializeReq 是 POST /install/initialize 的请求体（唯一写入口的入参）。
//
// 口令字段只出现在这里，且**禁止写入任何日志**（glog / 审计 / 错误详情均不得包含它）。
type InstallationInitializeReq struct {
	// AdminUsername 平台管理员用户名（必填，≤128）。
	AdminUsername string `json:"adminUsername" binding:"required,max=128"`
	// AdminPassword 管理员口令（必填）：由运维自选，服务端按 credential.ValidateStrength 强校验。
	AdminPassword string `json:"adminPassword" binding:"required,max=128"`
	// AdminEmail 与 AdminPhone 至少填一项（同 UserContactRequiredError 的口径）。
	AdminEmail string `json:"adminEmail" binding:"max=128"`
	AdminPhone string `json:"adminPhone" binding:"max=32"`
	// AdminName 管理员显示名（可选，留空取内置缺省）。
	AdminName string `json:"adminName" binding:"max=128"`
	// TenantName 平台租户名称（可选，留空取内置缺省）。
	TenantName string `json:"tenantName" binding:"max=128"`
	// Issuer 覆盖 OIDC issuer（可选，留空取本部署配置的 oidc.issuer）。
	// 存在的意义：内置客户端的回调/登出地址按 issuer 派生，反代拓扑下部署方需要显式指定。
	Issuer string `json:"issuer" binding:"max=256"`
}

// InstallationInitializeResp 是初始化成功的响应。
type InstallationInitializeResp struct {
	// Report 本次真实写入的明细（L1 只创建，故 action 恒为 created）。
	Report        InstallationReport `json:"report"`
	AdminUsername string             `json:"adminUsername"`
	// LoginURL 由 oidc.frontendLoginURL 派生，便于页面直接给出下一步入口。
	LoginURL string               `json:"loginURL"`
	Consoles InstallationConsoles `json:"consoles"`
}

// InstallationReport 是引导结果报告（对外形状，与 pkg/seed.Report 解耦：
// DTO 不直接暴露内部结构，避免内部字段调整变成对外破坏性变更）。
type InstallationReport struct {
	TenantID string               `json:"tenantId"`
	Changes  []InstallationChange `json:"changes"`
}

// InstallationChange 单条写入明细（entity/key/action）。
type InstallationChange struct {
	Entity string `json:"entity"`
	Key    string `json:"key"`
	Action string `json:"action"`
}
