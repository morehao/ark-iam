package model

import (
	"encoding/json"
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameUser = "tenant_user" // user 是 PostgreSQL 保留字；tenant_user 与 person（自然人）语义区分更清晰

type UserEntity struct {
	gormdao.BaseEntity
	TenantID     string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	PersonID     string          `gorm:"column:person_id;type:varchar(36);not null;default:'';comment:自然人ID"`
	Name         string          `gorm:"column:name;type:varchar(128);not null;default:'';comment:租户内姓名"`
	Avatar       string          `gorm:"column:avatar;type:varchar(2048);not null;default:'';comment:租户内头像URL"`
	Profile      json.RawMessage `gorm:"column:profile;type:json;not null;default:'{}';comment:租户内配置信息"`
	CustomData   json.RawMessage `gorm:"column:custom_data;type:json;not null;default:'{}';comment:租户内自定义数据"`
	IsSuspended  int8            `gorm:"column:is_suspended;type:smallint;not null;default:0;comment:是否挂起"`
	IsOwner      int8            `gorm:"column:is_owner;type:smallint;not null;default:0;comment:是否租户拥有者"`
	JoinedAt     *time.Time      `gorm:"column:joined_at;not null;default:CURRENT_TIMESTAMP;comment:加入租户时间"`
	LastSignInAt *time.Time      `gorm:"column:last_sign_in_at;comment:最后登录时间"`
	CreatedBy    string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy    string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy    string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (UserEntity) TableName() string {
	return TableNameUser
}

type UserEntityList []UserEntity

func (l UserEntityList) ToMap() map[string]UserEntity {
	m := make(map[string]UserEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
