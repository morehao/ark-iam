package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

const TableNameUser = "user"

type UserEntity struct {
	gorm.Model
	TenantID          uint            `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	Username          string          `gorm:"column:username;type:varchar(128);not null;default '';comment:用户名"`
	PrimaryEmail      string          `gorm:"column:primary_email;type:varchar(128);not null;default '';comment:主要邮箱"`
	PrimaryPhone      string          `gorm:"column:primary_phone;type:varchar(128);not null;default '';comment:主要手机号"`
	PasswordEncrypted string          `gorm:"column:password_encrypted;type:varchar(256);not null;default '';comment:加密密码"`
	PasswordMethod    string          `gorm:"column:password_method;type:varchar(32);not null;default '';comment:密码加密方式"`
	Name              string          `gorm:"column:name;type:varchar(128);not null;default '';comment:姓名"`
	Avatar            string          `gorm:"column:avatar;type:varchar(2048);not null;default '';comment:头像URL"`
	Profile           json.RawMessage `gorm:"column:profile;type:json;not null;default '{}';comment:配置信息"`
	ApplicationID     uint            `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:应用ID"`
	Identities        json.RawMessage `gorm:"column:identities;type:json;not null;default '{}';comment:第三方身份"`
	CustomData        json.RawMessage `gorm:"column:custom_data;type:json;not null;default '{}';comment:自定义数据"`
	IsSuspended       int8            `gorm:"column:is_suspended;type:tinyint(1);not null;default 0;comment:是否挂起"`
	LastSignInAt      *gorm.DeletedAt `gorm:"column:last_sign_in_at;comment:最后登录时间"`
	CreatedBy         uint            `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy         uint            `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy         uint            `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
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
