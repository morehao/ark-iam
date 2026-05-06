package objapplication

type ApplicationBaseInfo struct {
	TenantID     uint   `json:"tenantID" comment:"租户id"`
	Name         string `json:"name" comment:"应用名称"`
	Secret       string `json:"secret" comment:"应用密钥"`
	Description  string `json:"description" comment:"应用描述"`
	Type         string `json:"type" comment:"应用类型"`
	IsThirdParty int8   `json:"isThirdParty" comment:"是否第三方应用"`
}