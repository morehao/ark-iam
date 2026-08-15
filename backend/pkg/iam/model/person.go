package model

import (
	"encoding/json"
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNamePerson = "person"

type PersonEntity struct {
	gormdao.BaseEntity
	// Username/PrimaryEmail/PrimaryPhone 为可选全局标识，空值存 NULL：
	// 三者均有唯一索引，若空值存 '' 则仅允许一条空记录，后续创建无该标识的用户会撞唯一键。
	Username          *string         `gorm:"column:username;type:varchar(128);default:null;comment:全局用户名"`
	PrimaryEmail      *string         `gorm:"column:primary_email;type:varchar(128);default:null;comment:主要邮箱"`
	PrimaryPhone      *string         `gorm:"column:primary_phone;type:varchar(128);default:null;comment:主要手机号"`
	PasswordEncrypted string          `gorm:"column:password_encrypted;type:varchar(256);not null;default:'';comment:加密密码"`
	PasswordMethod    string          `gorm:"column:password_method;type:varchar(32);not null;default:'';comment:密码加密方式"`
	Name              string          `gorm:"column:name;type:varchar(128);not null;default:'';comment:姓名"`
	Avatar            string          `gorm:"column:avatar;type:varchar(2048);not null;default:'';comment:头像URL"`
	Profile           json.RawMessage `gorm:"column:profile;type:json;not null;default:'{}';comment:配置信息"`
	CustomData        json.RawMessage `gorm:"column:custom_data;type:json;not null;default:'{}';comment:自定义数据"`
	IsSuspended       bool            `gorm:"column:is_suspended;type:boolean;not null;default:false;comment:是否挂起"`
	LastSignInAt      *time.Time      `gorm:"column:last_sign_in_at;comment:最后登录时间"`
	CreatedBy         string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy         string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy         string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

// StrPtr 将空字符串转为 nil（NULL），非空返回指针，供 person 可选标识字段使用。
func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// DerefStr 解引用可空字符串，空/ nil 返回 ""。
func DerefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (PersonEntity) TableName() string {
	return TableNamePerson
}

type PersonEntityList []PersonEntity

func (l PersonEntityList) ToMap() map[string]PersonEntity {
	m := make(map[string]PersonEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
