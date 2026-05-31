package model

import (
	"gorm.io/gorm"
)

const TableNameMenu = "menu"

type MenuEntity struct {
	gorm.Model
	ApplicationID uint   `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:所属应用id"`
	ParentID     uint   `gorm:"column:parent_id;type:bigint unsigned;not null;default 0;comment:父菜单ID"`
	Name         string `gorm:"column:name;type:varchar(128);not null;default '';comment:菜单名称"`
	Code         string `gorm:"column:code;type:varchar(64);not null;default '';comment:菜单编码"`
	Path         string `gorm:"column:path;type:varchar(512);not null;default '';comment:菜单路径"`
	Icon         string `gorm:"column:icon;type:varchar(256);not null;default '';comment:菜单图标"`
	Sort         int    `gorm:"column:sort;type:int;not null;default 0;comment:排序"`
	Type         string `gorm:"column:type;type:varchar(32);not null;default '';comment:菜单类型"`
	Component    string `gorm:"column:component;type:varchar(256);not null;default '';comment:组件路径"`
	Redirect     string `gorm:"column:redirect;type:varchar(512);not null;default '';comment:重定向路径"`
	Hidden       int8   `gorm:"column:hidden;type:tinyint(1);not null;default 0;comment:是否隐藏"`
	ExternalLink int8   `gorm:"column:external_link;type:tinyint(1);not null;default 0;comment:是否外链"`
	KeepAlive    int8   `gorm:"column:keep_alive;type:tinyint(1);not null;default 0;comment:是否缓存"`
	Permission   string `gorm:"column:permission;type:varchar(128);not null;default '';comment:权限标识"`
	Status       string `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态"`
	CreatedBy    uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy    uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy    uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (MenuEntity) TableName() string {
	return TableNameMenu
}

type MenuEntityList []MenuEntity

func (l MenuEntityList) ToMap() map[uint]MenuEntity {
	m := make(map[uint]MenuEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}