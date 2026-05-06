package model

import (
	"gorm.io/gorm"
)

const TableNameRefreshToken = "refresh_token"

type RefreshTokenEntity struct {
	gorm.Model
	TenantID     uint          `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	UserID       uint          `gorm:"column:user_id;type:bigint unsigned;not null;default 0;comment:用户ID"`
	ApplicationID uint         `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:应用ID"`
	Token        string        `gorm:"column:token;type:varchar(256);not null;default '';comment:token哈希"`
	ExpiresAt    *gorm.DeletedAt `gorm:"column:expires_at;comment:过期时间"`
	RevokedAt    *gorm.DeletedAt `gorm:"column:revoked_at;comment:撤销时间"`
	CreatedBy    uint          `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy    uint          `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy    uint          `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
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