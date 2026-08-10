package model

import (
	"database/sql"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const TableNameApiKey = "api_key"

type ApiKeyEntity struct {
	gorm.Model
	TenantID   uint            `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	Name       string          `gorm:"column:name;type:varchar(128);not null;default '';comment:密钥名称"`
	KeyHash    string          `gorm:"column:key_hash;type:varchar(256);not null;default '';comment:密钥哈希"`
	KeyPrefix  string          `gorm:"column:key_prefix;type:varchar(16);not null;default '';comment:密钥前缀(前7位)"`
	Scope      json.RawMessage `gorm:"column:scope;type:json;not null;default '{}';comment:权限范围"`
	ExpiredAt  *time.Time      `gorm:"column:expired_at;type:datetime;comment:过期时间"`
	LastUsedAt sql.NullTime    `gorm:"column:last_used_at;type:datetime;comment:最后使用时间"`
	RevokedAt  *time.Time      `gorm:"column:revoked_at;type:datetime;comment:撤销时间"`
	CreatedBy  uint            `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy  uint            `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy  uint            `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}

func (ApiKeyEntity) TableName() string {
	return TableNameApiKey
}

type ApiKeyEntityList []ApiKeyEntity

func (l ApiKeyEntityList) ToMap() map[uint]ApiKeyEntity {
	m := make(map[uint]ApiKeyEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
