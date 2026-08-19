package objpermission

type MenuBaseInfo struct {
	AppID        string `json:"appID" form:"appID" binding:"required"` // 应用ID（菜单必须归属某个应用）
	ParentID     string `json:"parentID" form:"parentID"`              // 父菜单ID
	Name         string `json:"name" form:"name"`                      // 菜单名称
	Code         string `json:"code" form:"code"`                      // 菜单编码
	Path         string `json:"path" form:"path"`                      // 菜单路径
	Icon         string `json:"icon" form:"icon"`                      // 菜单图标
	Sort         int    `json:"sort" form:"sort"`                      // 排序
	Type         string `json:"type" form:"type"`                      // 菜单类型
	Visibility   string `json:"visibility" form:"visibility"`          // 可见性门槛(public/member/admin)
	Component    string `json:"component" form:"component"`            // 组件路径
	Redirect     string `json:"redirect" form:"redirect"`              // 重定向路径
	Hidden bool   `json:"hidden" form:"hidden"`                  // 是否隐藏
	ExternalLink bool   `json:"externalLink" form:"externalLink"`      // 是否外链
	KeepAlive bool   `json:"keepAlive" form:"keepAlive"`            // 是否缓存
	Permission   string `json:"permission" form:"permission"`          // 权限标识
	Status       string `json:"status" form:"status"`                  // 状态
}
