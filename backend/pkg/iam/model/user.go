package model

import (
	"encoding/json"
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

// TableNameUser 租户内用户表的物理表名。
// 领域实体保持 UserEntity（租户内用户，与 PersonEntity 自然人区分，贯穿 API/DTO/前端）；
// 因 user 是 PostgreSQL 保留字（DAO 拼接 user.xxx 会语法错误），物理表名采用 tenant_user，
// 由 UserEntity.TableName() 建立映射。
const TableNameUser = "tenant_user"

// UserType 租户账号类型。
type UserType string

// 账号类型取值（禁止硬编码）。
const (
	UserTypeMember  UserType = "member"  // 真实用户：person 映射的租户成员，可登录、可入组织
	UserTypeMachine UserType = "machine" // 服务账号：租户内机器主体，不可登录、不入组织，作为 API Key 归属主体
)

// IsReal 判断是否为真实用户（member）。
func (t UserType) IsReal() bool {
	return t == UserTypeMember
}

type UserEntity struct {
	gormdao.BaseEntity
	TenantID     string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	PersonID     string          `gorm:"column:person_id;type:varchar(36);not null;default:'';comment:自然人ID(服务账号恒空)"`
	UserType     UserType        `gorm:"column:user_type;type:varchar(16);not null;default:'member';comment:账号类型(member真实用户/machine服务账号)"`
	Name         string          `gorm:"column:name;type:varchar(128);not null;default:'';comment:租户内姓名/服务账号名称"`
	Description  string          `gorm:"column:description;type:varchar(256);not null;default:'';comment:描述(服务账号用途等)"`
	Avatar       string          `gorm:"column:avatar;type:varchar(2048);not null;default:'';comment:租户内头像URL"`
	Profile      json.RawMessage `gorm:"column:profile;type:json;not null;default:'{}';comment:租户内配置信息"`
	CustomData   json.RawMessage `gorm:"column:custom_data;type:json;not null;default:'{}';comment:租户内自定义数据"`
	IsSuspended  bool            `gorm:"column:is_suspended;type:boolean;not null;default:false;comment:是否挂起"`
	IsOwner      bool            `gorm:"column:is_owner;type:boolean;not null;default:false;comment:是否租户拥有者"`
	JoinedAt     *time.Time      `gorm:"column:joined_at;not null;default:CURRENT_TIMESTAMP;comment:加入租户时间"`
	LastSignInAt *time.Time      `gorm:"column:last_sign_in_at;comment:最后登录时间"`
	CreatedBy    string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy    string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy    string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (UserEntity) TableName() string {
	return TableNameUser
}

// IsMachine 判断租户账号是否为服务账号。
func (u *UserEntity) IsMachine() bool {
	return u != nil && UserType(u.UserType) == UserTypeMachine
}

// IsReal 判断租户账号是否为真实用户。
func (u *UserEntity) IsReal() bool {
	return u != nil && UserType(u.UserType).IsReal()
}

type UserEntityList []UserEntity

func (l UserEntityList) ToMap() map[string]UserEntity {
	m := make(map[string]UserEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
