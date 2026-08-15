package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameMenu = "menu"

type MenuEntity struct {
	gormdao.BaseEntity
	AppID        string `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:所属应用id"`
	ParentID     string `gorm:"column:parent_id;type:varchar(36);not null;default:'';comment:父菜单ID"`
	Name         string `gorm:"column:name;type:varchar(128);not null;default:'';comment:菜单名称"`
	Code         string `gorm:"column:code;type:varchar(64);not null;default:'';comment:菜单编码"`
	Path         string `gorm:"column:path;type:varchar(512);not null;default:'';comment:菜单路径"`
	Icon         string `gorm:"column:icon;type:varchar(256);not null;default:'';comment:菜单图标"`
	Sort         int    `gorm:"column:sort;type:int;not null;default:0;comment:排序"`
	Type         string `gorm:"column:type;type:varchar(32);not null;default:'';comment:菜单类型"`
	Component    string `gorm:"column:component;type:varchar(256);not null;default:'';comment:组件路径"`
	Redirect     string `gorm:"column:redirect;type:varchar(512);not null;default:'';comment:重定向路径"`
	Hidden       int8   `gorm:"column:hidden;type:smallint;not null;default:0;comment:是否隐藏"`
	ExternalLink int8   `gorm:"column:external_link;type:smallint;not null;default:0;comment:是否外链"`
	KeepAlive    int8   `gorm:"column:keep_alive;type:smallint;not null;default:0;comment:是否缓存"`
	Permission   string `gorm:"column:permission;type:varchar(128);not null;default:'';comment:权限标识"`
	Status       string `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态"`
	CreatedBy    string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy    string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy    string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
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
