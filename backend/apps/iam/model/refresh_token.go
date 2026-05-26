package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameRefreshToken = "refresh_token"

type RefreshTokenEntity struct {
	gorm.Model
	PersonID      uint       `gorm:"column:person_id;type:bigint unsigned;not null;default 0;comment:自然人ID"`
	TenantID      uint       `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	UserID        uint       `gorm:"column:user_id;type:bigint unsigned;not null;default 0;comment:用户ID"`
	ApplicationID uint       `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:应用ID"`
	SessionID     string     `gorm:"column:session_id;type:varchar(64);not null;default '';comment:会话ID"`
	Token         string     `gorm:"column:token;type:varchar(256);not null;default '';comment:token哈希"`
	ClientType    string     `gorm:"column:client_type;type:varchar(32);not null;default '';comment:客户端类型"`
	ClientIP      string     `gorm:"column:client_ip;type:varchar(64);not null;default '';comment:客户端IP"`
	UserAgent     string     `gorm:"column:user_agent;type:varchar(512);not null;default '';comment:用户代理信息"`
	ExpiredAt     *time.Time `gorm:"column:expired_at;type:datetime;comment:过期时间"`
	RevokedAt     *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间"`
	CreatedBy     uint       `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy     uint       `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy     uint       `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (RefreshTokenEntity) TableName() string {
	return TableNameRefreshToken
}

type RefreshTokenEntityList []RefreshTokenEntity

func (l RefreshTokenEntityList) ToMap() map[uint]RefreshTokenEntity {
	m := make(map[uint]RefreshTokenEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
