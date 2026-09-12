package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameMenu = "menu"

// MenuVisibility 菜单可见性门槛（单调：等级越高可见范围越窄）。
type MenuVisibility string

// 菜单可见性门槛取值（禁止硬编码）。
const (
	MenuVisibilityPublic MenuVisibility = "public" // 所有人可见（无门槛）
	MenuVisibilityMember MenuVisibility = "member" // 任意租户成员可见（登录即可）
	MenuVisibilityAdmin  MenuVisibility = "admin"  // 仅管理员角色可见（硬隔离）
)

// VisibilityRank 返回可见性门槛的序数（用于线性比较：public < member < admin）。
func (v MenuVisibility) VisibilityRank() int {
	switch v {
	case MenuVisibilityAdmin:
		return 3
	case MenuVisibilityMember:
		return 2
	default:
		return 1 // MenuVisibilityPublic
	}
}

// MenuType 菜单类型。
type MenuType string

// 菜单类型取值（禁止硬编码）。
const (
	MenuTypeDirectory MenuType = "directory" // 目录
	MenuTypeMenu      MenuType = "menu"      // 菜单
	MenuTypeButton    MenuType = "button"    // 按钮
)

// MenuStatus 菜单状态。
type MenuStatus string

// 菜单状态取值（禁止硬编码）。
const (
	MenuStatusEnable  MenuStatus = "enable"  // 启用
	MenuStatusDisable MenuStatus = "disable" // 停用
)

type MenuEntity struct {
	gormdao.BaseEntity
	AppID        string         `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id"`
	ParentID     string         `gorm:"column:parent_id;type:varchar(36);not null;default:'';comment:父菜单ID"`
	Name         string         `gorm:"column:name;type:varchar(128);not null;default:'';comment:菜单名称"`
	Code         string         `gorm:"column:code;type:varchar(64);not null;default:'';comment:菜单编码"`
	Path         string         `gorm:"column:path;type:varchar(512);not null;default:'';comment:菜单路径"`
	Icon         string         `gorm:"column:icon;type:varchar(256);not null;default:'';comment:菜单图标"`
	Sort         int            `gorm:"column:sort;type:int;not null;default:0;comment:排序"`
	Type         MenuType       `gorm:"column:type;type:varchar(32);not null;default:'';comment:菜单类型"`
	Visibility   MenuVisibility `gorm:"column:visibility;type:varchar(32);not null;default:'public';comment:可见性门槛(public/member/admin)" json:"visibility"`
	Component    string         `gorm:"column:component;type:varchar(256);not null;default:'';comment:组件路径"`
	Redirect     string         `gorm:"column:redirect;type:varchar(512);not null;default:'';comment:重定向路径"`
	Hidden       bool           `gorm:"column:hidden;type:boolean;not null;default:false;comment:是否隐藏"`
	ExternalLink bool           `gorm:"column:external_link;type:boolean;not null;default:false;comment:是否外链"`
	KeepAlive    bool           `gorm:"column:keep_alive;type:boolean;not null;default:false;comment:是否缓存"`
	Status       MenuStatus     `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态"`
	CreatedBy    string         `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy    string         `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy    string         `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (MenuEntity) TableName() string {
	return TableNameMenu
}

type MenuEntityList []MenuEntity

func (l MenuEntityList) ToMap() map[string]MenuEntity {
	m := make(map[string]MenuEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
