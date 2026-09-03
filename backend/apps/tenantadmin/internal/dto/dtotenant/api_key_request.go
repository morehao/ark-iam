package dtotenant

// 租户端 API 密钥请求 DTO（密钥一律归属服务账号）。

type ApiKeyPageListReq struct {
	Page          int    `json:"page" form:"page"`                   // 页码
	PageSize      int    `json:"pageSize" form:"pageSize"`           // 每页数量
	Name          string `json:"name" form:"name"`                   // 名称(模糊)
	MachineUserID string `json:"machineUserID" form:"machineUserID"` // 归属服务账号ID(空=租户全部密钥)
}

type ApiKeyCreateReq struct {
	Name          string `json:"name" binding:"required"`          // 密钥名称
	MachineUserID string `json:"machineUserID" binding:"required"` // 归属服务账号ID
	ExpiredAt     int64  `json:"expiredAt"`                        // 过期时间(unix秒,0=永不过期)
}

type ApiKeyRevokeReq struct {
	ApiKeyID string `json:"-" uri:"apiKeyID" binding:"required"` // API密钥ID
}

type ApiKeyDeleteReq struct {
	ApiKeyID string `json:"-" uri:"apiKeyID" binding:"required"` // API密钥ID
}
