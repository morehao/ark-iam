package objscope

type ScopeBaseInfo struct {
	TenantID   uint   `json:"tenantID" form:"tenantID"`     // 租户ID
	ResourceID uint   `json:"resourceID" form:"resourceID"` // 资源ID
	Name       string `json:"name" form:"name"`             // 权限名称
	Description string `json:"description" form:"description"` // 权限描述
}