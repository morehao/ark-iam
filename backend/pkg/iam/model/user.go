package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const TableNameUser = "user"

type UserEntity struct {
	gorm.Model
	TenantID     uint            `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	PersonID     uint            `gorm:"column:person_id;type:bigint unsigned;not null;default 0;comment:自然人ID"`
	Name         string          `gorm:"column:name;type:varchar(128);not null;default '';comment:租户内姓名"`
	Avatar       string          `gorm:"column:avatar;type:varchar(2048);not null;default '';comment:租户内头像URL"`
	Profile      json.RawMessage `gorm:"column:profile;type:json;not null;default '{}';comment:租户内配置信息"`
	CustomData   json.RawMessage `gorm:"column:custom_data;type:json;not null;default '{}';comment:租户内自定义数据"`
	IsSuspended  int8            `gorm:"column:is_suspended;type:tinyint(1);not null;default 0;comment:是否挂起"`
	IsOwner      int8            `gorm:"column:is_owner;type:tinyint(1);not null;default 0;comment:是否租户拥有者"`
	JoinedAt     *time.Time      `gorm:"column:joined_at;type:datetime;not null;default CURRENT_TIMESTAMP;comment:加入租户时间"`
	LastSignInAt *time.Time      `gorm:"column:last_sign_in_at;type:datetime;comment:最后登录时间"`
	CreatedBy    uint            `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy    uint            `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy    uint            `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (UserEntity) TableName() string {
	return TableNameUser
}

type UserEntityList []UserEntity

func (l UserEntityList) ToMap() map[uint]UserEntity {
	m := make(map[uint]UserEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
