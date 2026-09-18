package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNamePerson = "person"

// PasswordMethod 密码加密方式（字典值，落 person.password_method）。
type PasswordMethod string

// 密码加密方式取值（禁止硬编码）。
const (
	PasswordMethodBcrypt PasswordMethod = "bcrypt" // bcrypt 哈希
)

// PersonStatus 自然人可用性（person.status）。
// suspended = 禁止登录与签发令牌，语义对齐 tenant.status 的挂起。
type PersonStatus string

// 自然人状态取值（禁止硬编码）。
const (
	PersonStatusActive    PersonStatus = "active"
	PersonStatusSuspended PersonStatus = "suspended"
)

// PasswordStatus 自然人密码状态（person.password_status）。
type PasswordStatus string

// 密码状态取值（禁止硬编码）。
const (
	PasswordStatusNormal     PasswordStatus = "normal"
	PasswordStatusMustChange PasswordStatus = "must_change"
)

type PersonEntity struct {
	gormdao.BaseEntity
	// Username/PrimaryEmail/PrimaryPhone 为可选全局标识，空值存 NULL。
	// 唯一性由部分唯一索引保证（WHERE deleted_at IS NULL，见 automigrate.go
	// EnsurePartialUniqueIndexes），软删除记录不占用标识、也不与新增记录冲突。
	Username          *string        `gorm:"column:username;type:varchar(128);default:null;comment:全局用户名"`
	PrimaryEmail      *string        `gorm:"column:primary_email;type:varchar(128);default:null;comment:主要邮箱"`
	PrimaryPhone      *string        `gorm:"column:primary_phone;type:varchar(128);default:null;comment:主要手机号"`
	PasswordEncrypted string         `gorm:"column:password_encrypted;type:varchar(256);not null;default:'';comment:加密密码"`
	PasswordMethod    PasswordMethod `gorm:"column:password_method;type:varchar(32);not null;default:'';comment:密码加密方式"`
	// PasswordStatus 为 must_change 时，该自然人持临时密码（或密码刚被管理员重置），
	// 登录链路会在认证通过后拦截并强制其先设置新密码，改密成功前不签发 code / 不建会话。
	// 放在 person（而非 tenant_user）是因为密码是自然人全局凭据，同一个人在多个租户共享一套密码。
	PasswordStatus PasswordStatus `gorm:"column:password_status;type:varchar(16);not null;default:'normal';comment:密码状态(normal正常/must_change需改密)"`
	Name           string         `gorm:"column:name;type:varchar(128);not null;default:'';comment:姓名"`
	Avatar         string         `gorm:"column:avatar;type:varchar(2048);not null;default:'';comment:头像URL"`
	Status         PersonStatus   `gorm:"column:status;type:varchar(16);not null;default:'active';comment:状态(active正常/suspended挂起)"`
	LastSignInAt   *time.Time     `gorm:"column:last_sign_in_at;comment:最后登录时间"`
	CreatedBy      string         `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy      string         `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy      string         `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
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
